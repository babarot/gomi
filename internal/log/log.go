package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/babarot/gomi/internal/env"
	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

func New(ty string, attrs ...slog.Attr) *slog.Logger {
	var w io.Writer
	if file, err := os.OpenFile(env.GOMI_LOG_PATH, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		w = file
	} else {
		w = os.Stderr
	}

	handler := tint.NewHandler(w, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})
	switch ty {
	case "default":
		// use tint as default
	case "json":
		handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	}

	return slog.New(NewWrapHandler(
		handler,
		func() []slog.Attr {
			return attrs
		}),
	)
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
