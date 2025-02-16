package log

import (
	"io"
	"log/slog"

	"github.com/charmbracelet/log"
)

// applyToLoggers applies a given setting function to the provided loggers.
// If no logger is provided, the default logger is used.
func applyToLoggers(settingFunc func(*slog.Logger), loggers ...*slog.Logger) {
	if len(loggers) == 0 {
		loggers = append(loggers, Default())
	}

	for _, l := range loggers {
		settingFunc(l)
	}
}

// SetCallerFormatter sets the caller formatter.
func SetCallerFormatter(f CallerFormatter, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetCallerFormatter(f)
	}, loggers...)
}

// SetCallerOffset sets the caller offset.
func SetCallerOffset(offset int, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetCallerOffset(offset)
	}, loggers...)
}

// SetFormatter sets the formatter.
func SetFormatter(f Formatter, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetFormatter(f)
	}, loggers...)
}

// SetLevel sets the level.
func SetLevel(level Level, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetLevel(log.Level(level))
	}, loggers...)
}

// SetOutput sets the output destination.
func SetOutput(w io.Writer, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetOutput(w)
	}, loggers...)
}

// SetPrefix sets the prefix.
func SetPrefix(prefix string, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetPrefix(prefix)
	}, loggers...)
}

// WithPrefix returns a new logger with the given prefix.
func WithPrefix(l *slog.Logger, prefix string) *slog.Logger {
	SetPrefix(prefix, l)
	return l
}

// SetStyles sets the logger styles.
func SetStyles(s *Styles, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetStyles(s)
	}, loggers...)
}

// SetReportCaller sets whether to report caller location.
func SetReportCaller(report bool, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetReportCaller(report)
	}, loggers...)
}

// SetReportTimestamp sets whether to report timestamp.
func SetReportTimestamp(report bool, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetReportTimestamp(report)
	}, loggers...)
}

// SetTimeFormat sets the time format.
func SetTimeFormat(format string, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetTimeFormat(format)
	}, loggers...)
}

// SetTimeFunction sets the time function.
func SetTimeFunction(f TimeFunction, loggers ...*slog.Logger) {
	applyToLoggers(func(l *slog.Logger) {
		loggerHandler(l).SetTimeFunction(f)
	}, loggers...)
}
