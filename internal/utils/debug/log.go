package debug

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/utils/env"
	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

// TailLogs displays logs either by showing existing content or following new entries
func Logs(w io.Writer, cfg *config.LoggingConfig, live bool) error {
	if live {
		return tailLiveLogs(w, cfg)
	}
	return showExistingLogs(w, cfg)
}

// tailLiveLogs follows log entries in real-time
func tailLiveLogs(w io.Writer, cfg *config.LoggingConfig) error {
	// For live mode without logging enabled, return error
	if !cfg.Enabled {
		return fmt.Errorf("logging is not enabled in config: enable logging in config for live debugging")
	}

	shouldFollow := isatty.IsTerminal(os.Stdout.Fd())
	tailConfig := tail.Config{
		ReOpen: shouldFollow,
		Follow: shouldFollow,
		Poll:   true,
		Logger: tail.DiscardingLogger,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		},
	}

	// Create a channel to notify when tail starts
	started := make(chan struct{})

	// Start a goroutine to write the start message
	go func() {
		// Wait for tail to start
		<-started
		slog.Info("Live tail started")
	}()

	t, err := tail.TailFile(env.GOMI_LOG_PATH, tailConfig)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log file does not exist: try running some commands with logging enabled")
		}
		return err
	}

	// Notify that tail has started
	close(started)

	for line := range t.Lines {
		fmt.Fprintln(w, line.Text)
	}

	return nil
}

// showExistingLogs displays the current content of the log file
func showExistingLogs(w io.Writer, cfg *config.LoggingConfig) error {
	if _, err := os.Stat(env.GOMI_LOG_PATH); os.IsNotExist(err) {
		if !cfg.Enabled {
			return fmt.Errorf("logging is not enabled in config: enable logging to create log files")
		}
		return fmt.Errorf("no log file exists yet: try running some commands first")
	}

	f, err := os.Open(env.GOMI_LOG_PATH)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Fprintln(w, scanner.Text())
	}

	return scanner.Err()
}
