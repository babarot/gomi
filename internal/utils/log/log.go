// Package log provides a structured logger with context support.
package log

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Logger is an alias for [slog.Logger].
type Logger = slog.Logger

var (
	defaultStylesOnce struct {
		sync.Once
		s *log.Styles
	}

	defaultOnce struct {
		sync.Once
		l atomic.Pointer[slog.Logger]
	}
)

// DefaultStyles returns the default styles.
// It applies custom styles to the [log.DefaultStyles].
func DefaultStyles() *Styles {
	defaultStylesOnce.Do(func() {
		defaultStylesOnce.s = log.DefaultStyles()

		for _, level := range []struct {
			Level    Level
			MaxWidth int
			Style    lipgloss.Style
		}{
			{
				Level:    DebugLevel,
				MaxWidth: 5,
				Style:    lipgloss.NewStyle().Foreground(lipgloss.Color("#808080")), // Gray
			},
			{
				Level:    InfoLevel,
				MaxWidth: 5,
				Style:    lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF")), // Blue
			},
			{
				Level:    WarnLevel,
				MaxWidth: 5,
				Style:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")), // Yellow
			},
			{
				Level:    ErrorLevel,
				MaxWidth: 5,
				Style:    lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")), // Red
			},
			{
				Level:    FatalLevel,
				MaxWidth: 5,
				Style: lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FF0000")).
					Background(lipgloss.Color("#000000")).
					Bold(true), // Red on Black, Bold
			},
			{
				Level:    ImportantLevel,
				MaxWidth: 9,
				Style: lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FF5F87")).
					Background(lipgloss.Color("#3A3A3A")).
					Bold(true), // Custom Important style
			},
		} {
			levelStr := strings.ToUpper(level.Level.String())
			// add padding to keep MaxWidth
			if len(levelStr) < level.MaxWidth {
				levelStr = levelStr + strings.Repeat(" ", level.MaxWidth-len(levelStr))
			}
			defaultStylesOnce.s.Levels[level.Level] = level.Style.SetString(levelStr)
		}
	})

	return defaultStylesOnce.s
}

// Default returns the default logger.
func Default() *slog.Logger {
	defaultOnce.Do(func() {
		if defaultOnce.l.Load() != nil {
			return
		}
		defaultOnce.l.Store(New(AsDefault()))
	})
	return defaultOnce.l.Load()
}

// handler returns the default logger's handler.
func handler() *log.Logger {
	return loggerHandler(Default())
}

// loggerHandler returns the logger's handler.
func loggerHandler(l *slog.Logger) *log.Logger {
	return l.Handler().(*log.Logger)
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	return &Options{
		LogOptions: &LogOptions{
			Level: log.InfoLevel,
		},
		OutputFunc: nil,
		Writer:     os.Stderr,
		Styles:     DefaultStyles(),
	}
}

// New creates a new logger with the given options.
func New(opts ...Option) *slog.Logger {
	o := DefaultOptions()
	o.Apply(opts...)

	// Create writer if OutputFunc is set
	if o.OutputFunc != nil {
		if w, err := o.OutputFunc(); err == nil {
			o.Writer = w
		} else {
			o.Writer = os.Stderr
		}
	}

	handler := log.NewWithOptions(o.Writer, *o.LogOptions)

	if o.Styles != nil {
		handler.SetStyles(o.Styles)
	}

	l := slog.New(handler)

	if o.Default {
		log.SetDefault(handler)
		slog.SetDefault(l)
		defaultOnce.l.Store(l)
	}

	return l
}

func Debug(msg string, args ...any) {
	handler().Debug(msg, args...)
}

func Debugf(format string, args ...any) {
	handler().Debugf(format, args...)
}

func Info(msg string, args ...any) {
	handler().Info(msg, args...)
}

func Infof(format string, args ...any) {
	handler().Infof(format, args...)
}

func Warn(msg string, args ...any) {
	handler().Warn(msg, args...)
}

func Warnf(format string, args ...any) {
	handler().Warnf(format, args...)
}

func Error(msg string, args ...any) {
	handler().Error(msg, args...)
}

func Errorf(format string, args ...any) {
	handler().Errorf(format, args...)
}

func Fatal(msg any, keyvals ...any) {
	handler().Fatal(msg, keyvals...)
}

func Fatalf(format string, args ...any) {
	handler().Fatalf(format, args...)
}

func Important(msg string, args ...any) {
	handler().Log(ImportantLevel, msg, args...)
}

func Importantf(format string, args ...any) {
	handler().Logf(ImportantLevel, format, args...)
}

func Print(msg string, args ...any) {
	handler().Print(msg, args...)
}

func Log(level Level, msg string, args ...any) {
	handler().Log(level, msg, args...)
}

func Logf(level Level, format string, args ...any) {
	handler().Logf(level, format, args...)
}
