package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/env"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/legacy"
	"github.com/babarot/gomi/internal/trash/xdg"
	"github.com/babarot/gomi/internal/utils/debug"
	"github.com/charmbracelet/log"
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
	config  config.Config
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

	logDir := filepath.Dir(env.GOMI_LOG_PATH)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			return err
		}
	}

	var w io.Writer
	if file, err := os.OpenFile(env.GOMI_LOG_PATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		w = file
	} else {
		w = os.Stderr
	}

	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    true,
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
		Level:           log.DebugLevel,
		Formatter: func() log.Formatter {
			if strings.ToLower(opt.Meta.Debug) == "json" {
				return log.JSONFormatter
			}
			return log.TextFormatter
		}(),
	})
	logger.SetOutput(w)
	logger.With("run_id", runID())
	slog.SetDefault(slog.New(logger))

	defer slog.Debug("main function finished\n\n\n")
	slog.Debug("main function started", "version", v.Version, "revision", v.Revision, "buildDate", v.BuildDate)

	cfg, err := config.Parse(opt.Config)
	if err != nil {
		return err
	}

	// Initialize trash configuration
	trashConfig := trash.Config{
		Type:               trash.StorageTypeXDG,
		HomeTrashDir:       cfg.Core.TrashDir,
		EnableHomeFallback: cfg.Core.HomeFallback,
		UseXDG:             cfg.Core.UseXDG,
		Verbose:            cfg.Core.Verbose,
		History:            cfg.History,
		TrashDir:           cfg.Core.TrashDir,
		RunID:              runID(),
	}

	if !cfg.Core.UseXDG {
		trashConfig.Type = trash.StorageTypeLegacy
	}

	// Initialize storage manager with appropriate implementations
	var managerOpts []trash.ManagerOption

	// Always add the primary storage based on configuration
	if cfg.Core.UseXDG {
		managerOpts = append(managerOpts, trash.WithStorage(xdg.NewStorage))
		if exist, _ := trash.DetectExistingLegacy(); exist {
			managerOpts = append(managerOpts, trash.WithStorage(legacy.NewStorage))
		}
	} else {
		managerOpts = append(managerOpts, trash.WithStorage(legacy.NewStorage))
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
