package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/adrg/xdg"
	"github.com/mattn/go-isatty"
	"github.com/nxadm/tail"
)

var (
	AppName    string
	EnvLogPath string
)

func New(attrs ...slog.Attr) *slog.Logger {
	fp := getLogPath(EnvLogPath)

	var w io.Writer
	if file, err := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
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
	fp := getLogPath(EnvLogPath)

	shouldFollow := isatty.IsTerminal(os.Stdout.Fd())
	t, err := tail.TailFile(fp, tail.Config{Follow: shouldFollow, ReOpen: shouldFollow})
	if err != nil {
		return err
	}
	for line := range t.Lines {
		fmt.Fprintln(w, line.Text)
	}
	return nil
}

func getLogPath(env string) string {
	fp, ok := os.LookupEnv(env)
	if !ok {
		var err error
		fp, err = xdg.CacheFile(fmt.Sprintf("%s/log", AppName))
		if err != nil {
			fp = fmt.Sprintf("%s.log", AppName)
		}
	}
	return fp
}
