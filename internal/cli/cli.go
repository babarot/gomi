package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/legacy"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/utils/debug"
	"github.com/babarot/gomi/internal/utils/env"
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
	Debug   string `long:"debug" description:"View debug logs (default: \"full\")" optional-value:"full" optional:"yes" choice:"full" choice:"live"`
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

func Run(v Version) error {
	var opt Option
	parser := flags.NewParser(&opt, flags.Default)
	parser.Name = v.AppName
	parser.Usage = "[-b | files...]"
	args, err := parser.Parse()
	if err != nil {
		if flags.WroteHelp(err) {
			return nil
		}
		return err
	}

	_ = log.New(
		log.UseLevel(log.DebugLevel),
		log.UseOutputPath(env.GOMI_LOG_PATH),
		log.UseTimeFormat(time.Kitchen),
		log.UseReportTimestamp(true),
		log.UseReportCaller(true),
		log.AsDefault(), // seamlessly integrate with log/slog
	)

	defer slog.Debug("main function finished\n\n\n")
	slog.Debug("main function started", "version", v.Version, "revision", v.Revision, "buildDate", v.BuildDate)

	cfg, err := config.Load(opt.Config)
	if err != nil {
		return err
	}
	if cfg == nil {
		// NOTE: fallback to default config?
		return errors.New("panic when parsing config")
	}

	// Initialize trash configuration
	trashConfig := trash.Config{
		Strategy: trash.Strategy(cfg.Core.Trash.Strategy),
		// TODO: HomeFallback: cfg.Core.HomeFallback,
		History: cfg.History,
		GomiDir: cfg.Core.Trash.GomiDir,
		RunID:   runID(),
	}

	// Initialize storage manager with appropriate implementations
	var managerOpts []trash.ManagerOption

	// Always add the primary storage based on configuration
	switch trashConfig.Strategy {
	case trash.StrategyXDG:
		// Force XDG only
		managerOpts = append(managerOpts, trash.WithStorage(xdg.NewStorage))
	case trash.StrategyLegacy:
		// Force Legacy only
		managerOpts = append(managerOpts, trash.WithStorage(legacy.NewStorage))
	case trash.StrategyAuto:
		// Default to XDG with optional legacy fallback
		managerOpts = append(managerOpts, trash.WithStorage(xdg.NewStorage))
		if exist, err := trash.IsExistLegacy(); err != nil {
			log.Error("failed to check if legacy storage exists", "error", err)
		} else if exist {
			managerOpts = append(managerOpts, trash.WithStorage(legacy.NewStorage))
		}
	default:
		// Invalid strategy, default to XDG
		log.Warn("invalid trash strategy, defaulting to XDG", "strategy", cfg.Core.Trash.Strategy)
		managerOpts = append(managerOpts, trash.WithStorage(xdg.NewStorage))
	}

	manager, err := trash.NewManager(trashConfig, managerOpts...)
	if err != nil {
		return fmt.Errorf("failed to initialize storage manager: %w", err)
	}

	cli := CLI{
		version: v,
		option:  opt,
		config:  cfg,
		runID:   runID(),
		manager: manager,
	}

	if err := cli.Run(args); err != nil {
		log.Error("exit", "error", fmt.Errorf("cli.run failed: %w", err))
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

	default:
		switch c.option.Meta.Debug {
		case "live":
			return debug.Logs(os.Stdout, true)
		case "full":
			return debug.Logs(os.Stdout, false)
		}
		return c.Put(args)
	}
}
