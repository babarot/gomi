package debug

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

const (
	LiveMode string = "live"
	FullMode string = "full"
)

var (
	ErrLogFileNotFound = errors.New("no log file exists yet")
)

func Logs(w io.Writer, path string, live bool) error {
	if live {
		return tailLiveLogs(w, path)
	}
	return showExistingLogs(w, path)
}

// tailLiveLogs follows log entries in real-time
func tailLiveLogs(w io.Writer, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ErrLogFileNotFound
	}

	// Create a channel to notify when tail starts
	started := make(chan struct{})

	// Start a goroutine to write the start message
	go func() {
		// Wait for tail to start
		<-started
		slog.Info("Live tail started")
	}()

	shouldFollow := isatty.IsTerminal(os.Stdout.Fd())

	t, err := tail.TailFile(path, tail.Config{
		ReOpen: shouldFollow,
		Follow: shouldFollow,
		Poll:   true,
		Logger: tail.DiscardingLogger,
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd,
		},
	})
	if err != nil {
		if os.IsNotExist(err) {
			return ErrLogFileNotFound
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
func showExistingLogs(w io.Writer, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ErrLogFileNotFound
	}

	f, err := os.Open(path)
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
