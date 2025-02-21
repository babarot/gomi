package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/legacy"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/utils/debug"
	"github.com/babarot/gomi/internal/utils/log"
	"github.com/jessevdk/go-flags"
	"github.com/rs/xid"
)

type Option struct {
	Restore bool   `short:"b" long:"restore" description:"Restore deleted file"`
	Config  string `long:"config" description:"Path to config file" default:""`

	Meta MetaOption `group:"Meta Options"`
	Rm   RmOption   `group:"Compatible (rm) Options"`
}

type MetaOption struct {
	Version bool   `short:"V" long:"version" description:"Show version"`
	Debug   string `long:"debug" description:"View debug logs" optional-value:"full" optional:"yes" choice:"full" choice:"live"`
}

// RmOption provides compatibility with rm command options
// https://man7.org/linux/man-pages/man1/rm.1.html
type RmOption struct {
	Interactive bool `short:"i" description:"(dummy) prompt before every removal"`
	Recursive   bool `short:"r" long:"recursive" description:"(dummy) remove directories and their contents recursively"`
	Recursive2  bool `short:"R" description:"(dummy) same as -r"`
	Force       bool `short:"f" long:"force" description:"(dummy) ignore nonexistent files, never prompt"`
	Directory   bool `short:"d" long:"dir" description:"(dummy) remove empty directories"`
	Verbose     bool `short:"v" long:"verbose" description:"(dummy) explain what is being done"`
}

type CLI struct {
	version Version
	option  Option
	config  *config.Config
	runID   string
	manager *trash.Manager
}

var runID = sync.OnceValue(func() string {
	id := xid.New().String()
	return id
})

// Run is the main entry point for the CLI
func Run(v Version) error {
	opt, args, err := parseOptions(v)
	if err != nil {
		return err
	}
	if opt == nil {
		return nil // help was shown
	}

	cfg, err := config.Load(opt.Config)
	if err != nil {
		return err
	}
	if cfg == nil {
		// NOTE: fallback to default config?
		return errors.New("panic when parsing config")
	}

	if err := setLogger(cfg); err != nil {
		return err
	}

	manager, err := newTrashManager(cfg)
	if err != nil {
		return err
	}

	// Log startup information
	slog.Debug("main function started",
		"version", v.Version,
		"revision", v.Revision,
		"buildDate", v.BuildDate)
	defer slog.Debug("main function finished")

	cli := CLI{
		version: v,
		option:  *opt,
		config:  cfg,
		runID:   runID(),
		manager: manager,
	}

	if err := cli.Run(args); err != nil {
		slog.Error("exit", "error", fmt.Errorf("cli.run failed: %w", err))
		return err
	}

	return nil
}

func (c CLI) Run(args []string) error {
	switch {
	case c.option.Meta.Version:
		fmt.Fprint(os.Stdout, c.version.Print())
		return nil

	case c.option.Restore:
		return c.Restore()

	case c.option.Meta.Debug != "":
		return debug.Logs(os.Stdout, &c.config.Core.Logging, c.option.Meta.Debug == "live")

	default:
		return c.Put(args)
	}
}

// parseOptions parses and returns command line options
func parseOptions(v Version) (*Option, []string, error) {
	var opt Option
	parser := flags.NewParser(&opt, flags.Default)
	parser.Name = v.AppName
	parser.Usage = "[-b | files...]"

	args, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	// On Windows, the shell does not expand wildcards,
	// so the application must handle them.
	if runtime.GOOS == "windows" {
		args = expandWindowsPaths(args)
	}

	return &opt, args, nil
}

// expandWindowsPaths expands file paths with wildcards on Windows
func expandWindowsPaths(args []string) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err == nil && len(matches) > 0 {
			expanded = append(expanded, matches...)
		} else {
			expanded = append(expanded, arg)
		}
	}
	return expanded
}

// setLogger sets up the logging system based on configuration
func setLogger(cfg *config.Config) error {
	var logWriter io.Writer = io.Discard

	if cfg.Core.Logging.Enabled {
		writer, err := log.NewRotateWriter(&cfg.Core.Logging)
		if err != nil {
			return err
		}
		logWriter = writer
	}

	logLevel := determineLogLevel(cfg.Core.Logging.Level)

	_ = log.New(
		log.UseLevel(logLevel),
		log.UseOutput(logWriter),
		log.UseTimeFormat(time.Kitchen),
		log.UseReportTimestamp(true),
		log.UseReportCaller(true),
		log.AsDefault(), // seamlessly integrate with log/slog
	)

	return nil
}

// determineLogLevel converts string log level to log.Level
func determineLogLevel(level string) log.Level {
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.DebugLevel
	}
}

// newTrashManager creates and configures the trash manager
func newTrashManager(cfg *config.Config) (*trash.Manager, error) {
	trashConfig := trash.Config{
		Strategy:     trash.Strategy(cfg.Core.Trash.Strategy),
		HomeFallback: cfg.Core.HomeFallback,
		History:      cfg.History,
		GomiDir:      cfg.Core.Trash.GomiDir,
		RunID:        runID(), // for backward compatibility (legacy strategy)
	}

	var opts []trash.ManagerOption

	switch strategy := trashConfig.Strategy; strategy {
	case trash.StrategyXDG:
		// Force XDG only
		opts = append(opts, trash.WithStorage(xdg.NewStorage))

	case trash.StrategyLegacy:
		// Force Legacy only
		opts = append(opts, trash.WithStorage(legacy.NewStorage))

	case trash.StrategyAuto:
		// Default to XDG with optional legacy fallback
		opts = append(opts, trash.WithStorage(xdg.NewStorage))
		if exist, err := trash.IsExistLegacy(); err == nil && exist {
			opts = append(opts, trash.WithStorage(legacy.NewStorage))
		}

	default:
		slog.Warn("invalid trash strategy, defaulting to XDG", "strategy", strategy)
		opts = append(opts, trash.WithStorage(xdg.NewStorage))
	}

	manager, err := trash.NewManager(trashConfig, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage manager: %w", err)
	}

	return manager, nil
}
