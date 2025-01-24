package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/babarot/gomi/internal/env"
	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

func New(attrs ...slog.Attr) *slog.Logger {
	var w io.Writer
	if file, err := os.OpenFile(env.GOMI_LOG_PATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		w = file
	} else {
		w = os.Stderr
	}

	handler := NewWrapHandler(
		slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		func() []slog.Attr {
			return attrs
		})

	return slog.New(handler)
}

func Follow(w io.Writer) error {
	shouldFollow := isatty.IsTerminal(os.Stdout.Fd())
	t, err := tail.TailFile(env.GOMI_LOG_PATH, tail.Config{Follow: shouldFollow, ReOpen: shouldFollow})
	if err != nil {
		return err
	}
	for line := range t.Lines {
		fmt.Fprintln(w, line.Text)
	}
	return nil
}
