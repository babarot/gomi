package log

import (
	"fmt"
	"log/slog"
	"sync"

	charmlog "github.com/charmbracelet/log"
)

// New creates a new logger with the given options
func New(opts ...Option) (*slog.Logger, error) {
	o := DefaultOptions()
	// Apply options and handle potential errors
	if err := o.Apply(opts...); err != nil {
		return nil, fmt.Errorf("failed to apply options: %w", err)
	}

	// Create and configure the handler
	handler := charmlog.NewWithOptions(o.Writer, o.Options)
	handler.SetStyles(o.Styles) // Always set styles to ensure level definitions

	// Create the logger
	logger := slog.New(handler)

	// Set as default if requested
	if o.Default {
		charmlog.SetDefault(handler)
		slog.SetDefault(logger)
		defaultLogger.Store(logger)
	}

	return logger, nil
}

// Default returns the default logger instance
func Default() *slog.Logger {
	l, _ := New(AsDefault())
	defaultLoggerOnce.Do(func() {
		if defaultLogger.Load() == nil {
			defaultLogger.Store(l)
		}
	})
	return defaultLogger.Load()
}

// Reset resets all global state (useful for testing)
func Reset() {
	defaultStylesOnce = sync.Once{}
	defaultStyles.Store(nil)
	defaultLoggerOnce = sync.Once{}
	defaultLogger.Store(nil)
}

func handler() *charmlog.Logger {
	return loggerHandler(Default())
}

func loggerHandler(l *slog.Logger) *charmlog.Logger {
	return l.Handler().(*charmlog.Logger)
}

// Logging methods
func Debug(msg string, args ...any)     { handler().Debug(msg, args...) }
func Info(msg string, args ...any)      { handler().Info(msg, args...) }
func Warn(msg string, args ...any)      { handler().Warn(msg, args...) }
func Error(msg string, args ...any)     { handler().Error(msg, args...) }
func Fatal(msg any, args ...any)        { handler().Fatal(msg, args...) }
func Important(msg string, args ...any) { handler().Log(ImportantLevel, msg, args...) }

func Debugf(format string, args ...any)     { handler().Debugf(format, args...) }
func Infof(format string, args ...any)      { handler().Infof(format, args...) }
func Warnf(format string, args ...any)      { handler().Warnf(format, args...) }
func Errorf(format string, args ...any)     { handler().Errorf(format, args...) }
func Fatalf(format string, args ...any)     { handler().Fatalf(format, args...) }
func Importantf(format string, args ...any) { handler().Logf(ImportantLevel, format, args...) }
