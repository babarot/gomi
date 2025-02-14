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
	"github.com/babarot/gomi/internal/trash/core"
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

// This should be not conflicts with app option
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
	config  config.Config
	runID   string
	// history history.History
	// storage core.Storage // Storage implementation (XDG or Legacy)
	storage *trash.Manager
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
			// TODO: fix this
			// json is no longer valid argument so doesnt work anymore.
			if strings.ToLower(opt.Meta.Debug) == "json" {
				return log.JSONFormatter
			}
			return log.TextFormatter
		}(),
	})

	logger.SetOutput(w)
	logger.With("run_id", runID())
	slog.SetDefault(slog.New(logger))

	defer slog.Debug("main function finished\n\n\n\n")
	slog.Debug("main function started", "version", v.Version, "revision", v.Revision, "buildDate", v.BuildDate)

	cfg, err := config.Parse(opt.Config)
	if err != nil {
		return err
	}

	// Initialize appropriate storage using factory
	// storageType := trash.StorageTypeXDG
	// if !cfg.Core.UseXDG {
	// 	storageType = trash.StorageTypeLegacy
	// }

	trashConfig := core.Config{
		Type:               core.StorageTypeXDG,
		UseXDG:             cfg.Core.UseXDG,
		EnableHomeFallback: cfg.Core.HomeFallback,
		HomeTrashDir:       cfg.Core.TrashDir,
		Verbose:            cfg.Core.Verbose,
		// TODO:
		TrashDir: cfg.Core.TrashDir,
		History:  cfg.History,
		RunID:    runID(),
	}
	if !cfg.Core.UseXDG {
		trashConfig.Type = core.StorageTypeLegacy
	}
	storageManager, err := trash.NewManager(trashConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize storage manager: %w", err)
	}

	// // Initialize appropriate storage based on configuration
	// storageConfig := &core.Config{
	// 	HomeTrashDir:       cfg.Core.TrashDir,
	// 	EnableHomeFallback: cfg.Core.HomeFallback,
	// 	Verbose:            cfg.Core.Verbose,
	// }
	//
	// var storage core.Storage
	// if cfg.Core.UseXDG {
	// 	storage, err = xdg.NewStorage(storageConfig)
	// 	// if legacy, _ := trash.DetectLegacy(); legacy {
	// 	// 	// storage, err = legacy.NewStorage(storageConfig)
	// 	// }
	// } else {
	// 	storage, err = legacy.NewStorage(storageConfig)
	// }
	// if err != nil {
	// 	return fmt.Errorf("failed to initialize storage: %w", err)
	// }

	cli := CLI{
		version: v,
		option:  opt,
		config:  cfg,
		runID:   runID(),
		storage: storageManager,
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
