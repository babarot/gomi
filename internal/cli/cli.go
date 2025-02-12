package cli

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/debug"
	"github.com/babarot/gomi/internal/env"
	"github.com/babarot/gomi/internal/history"
	"github.com/babarot/gomi/internal/ui"
	"github.com/charmbracelet/log"
	"github.com/jessevdk/go-flags"
	"github.com/rs/xid"
	"golang.org/x/sync/errgroup"
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
	history history.History
	runID   string
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

	defer slog.Debug("main function finished")
	slog.Debug("main function started", "version", v.Version, "revision", v.Revision, "buildDate", v.BuildDate)

	cfg, err := config.Parse(opt.Config)
	if err != nil {
		return err
	}

	cli := CLI{
		version: v,
		option:  opt,
		config:  cfg,
		history: history.New(cfg.Core.TrashDir, cfg.History),
		runID:   runID(),
	}

	if err := cli.Run(args); err != nil {
		slog.Error("exit", "error", fmt.Errorf("cli.run failed: %w", err))
		return err
	}
	return nil
}

func (c CLI) Run(args []string) error {
	if err := c.history.Open(); err != nil {
		return err
	}

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

func (c CLI) Restore() error {
	slog.Debug("cli.restore started")
	defer slog.Debug("cli.restore finished")

	if len(c.history.Files) == 0 {
		fmt.Printf("The history is empty. Let's try deleting a file first\n")
		return nil
	}

	files := c.history.Filter()
	if len(files) == 0 {
		fmt.Printf("Could not find any files to display. The history may be empty, or the history.exclude conditions may be too strict\n")
		return nil
	}

	files, err := ui.RenderList(files, c.config.UI)
	if err != nil {
		return err
	}

	var deletedFiles []history.File
	var errs []error

	defer func() {
		for _, file := range deletedFiles {
			err := c.history.Remove(file)
			if err != nil {
				slog.Error("removing from history failed", "file", file.Name, "error", err)
			}
			if c.config.Core.Restore.Verbose {
				fmt.Printf("restored %s to %s\n", file.Name, file.From)
			}
		}
	}()

	for _, file := range files {
		if _, err := os.Stat(file.From); err == nil {
			newName, err := ui.InputFilename(file)
			if err != nil {
				if errors.Is(err, ui.ErrInputCanceled) {
					if c.config.Core.Restore.Verbose {
						if newName == "" {
							fmt.Printf("canceled!\n")
						} else {
							fmt.Printf("you're inputting %q but it's canceled!\n", newName)
						}
					}
					continue
				}
				errs = append(errs, err)
				continue
			}
			// Update with new name
			file.From = filepath.Join(filepath.Dir(file.From), newName)
		}
		allowed := func() bool {
			if _, err := os.Stat(file.From); !os.IsNotExist(err) {
				yes := ui.Confirm(
					fmt.Sprintf("Caution! The same name already exists. Even so okay to restore? %s",
						filepath.Base(file.From)))
				if yes {
					return true
				}
				if c.config.Core.Restore.Verbose {
					fmt.Printf("Replied no, canceled!\n")
				}
				return false
			}
			if c.config.Core.Restore.Confirm {
				yes := ui.Confirm(fmt.Sprintf("OK to put back? %s", filepath.Base(file.From)))
				if yes {
					return true
				}
				if c.config.Core.Restore.Verbose {
					fmt.Printf("Replied no, canceled!\n")
				}
				return false
			}
			return true
		}
		if !allowed() {
			continue
		}
		err := move(file.To, file.From)
		if err != nil {
			errs = append(errs, err)
			slog.Error("failed to restore! file would not be deleted from history file", "error", err)
			continue
		}
		deletedFiles = append(deletedFiles, file)
	}

	// respect https://github.com/hashicorp/go-multierror
	if len(errs) > 0 {
		lines := []string{fmt.Sprintf("%d errors occurred:", len(errs))}
		for _, err := range errs {
			lines = append(lines, fmt.Sprintf("\t* %s", err))
		}
		lines = append(lines, "\n")
		return errors.New(strings.Join(lines, "\n"))
	}
	return nil
}

func (c CLI) Put(args []string) error {
	slog.Debug("cli.put started")
	defer slog.Debug("cli.put finished")

	if len(args) == 0 {
		return errors.New("too few arguments")
	}

	files := make([]history.File, len(args))
	var eg errgroup.Group
	var mu sync.Mutex // Mutex to synchronize writes to files

	for i, arg := range args {
		i, arg := i, arg // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			_, err := os.Stat(arg)
			if os.IsNotExist(err) {
				return fmt.Errorf("%s: no such file or directory", arg)
			}
			file, err := c.history.FileInfo(c.runID, arg)
			if err != nil {
				return err
			}

			// Lock before modifying the shared 'files' slice
			mu.Lock()
			files[i] = file
			mu.Unlock()

			return move(file.From, file.To)
		})
	}

	// Save the files after all tasks are done
	defer func() {
		err := c.history.Update(files)
		if err != nil {
			slog.Error("failed to update history", "error", err)
		}
	}()

	// Wait for all goroutines to complete
	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
