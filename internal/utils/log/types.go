package log

import (
	"time"

	charmlog "github.com/charmbracelet/log"
)

type (
	Logger          = charmlog.Logger
	Level           = charmlog.Level
	Styles          = charmlog.Styles
	CallerFormatter = charmlog.CallerFormatter
	Formatter       = charmlog.Formatter
	TimeFunction    = func(time.Time) time.Time
)

const (
	DebugLevel     = charmlog.DebugLevel
	InfoLevel      = charmlog.InfoLevel
	WarnLevel      = charmlog.WarnLevel
	ErrorLevel     = charmlog.ErrorLevel
	FatalLevel     = charmlog.FatalLevel
	ImportantLevel = WarnLevel + 1
)

// Caller Formatters
var (
	ShortCallerFormatter = charmlog.ShortCallerFormatter
	LongCallerFormatter  = charmlog.LongCallerFormatter
)

// Formatters
const (
	TextFormatter   = charmlog.TextFormatter
	JSONFormatter   = charmlog.JSONFormatter
	LogfmtFormatter = charmlog.LogfmtFormatter
)

// LogLevel returns the string representation of the level
func LogLevelString(l Level) string {
	switch l {
	case ImportantLevel:
		return " IMPORTANT "
	default:
		return charmlog.Level(l).String()
	}
}
