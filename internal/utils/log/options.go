package log

import (
	"io"
	"os"

	charmlog "github.com/charmbracelet/log"
)

// Options represents logger configuration options
type Options struct {
	charmlog.Options // embed Options instead of pointer
	Writer           io.Writer
	Styles           *Styles
	Default          bool
	OutputFunc       func() (io.Writer, error)
}

// DefaultOptions returns the default logger options
func DefaultOptions() *Options {
	return &Options{
		Options: charmlog.Options{
			Level:           InfoLevel,
			ReportCaller:    false,
			ReportTimestamp: false,
		},
		Writer: os.Stderr,
		Styles: DefaultStyles(),
	}
}

// Apply applies the given options
func (o *Options) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Option func(*Options)

func UseLevel(l Level) Option {
	return func(o *Options) {
		o.Level = l
	}
}

func UseOutput(w io.Writer) Option {
	return func(o *Options) {
		o.Writer = w
	}
}

func UseOutputFunc(f func() (io.Writer, error)) Option {
	return func(o *Options) {
		o.OutputFunc = f
	}
}

func UseOutputPath(path string) Option {
	return UseOutputFunc(func() (io.Writer, error) {
		if path == "" {
			return os.Stderr, nil
		}
		return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	})
}

func UseReportCaller(report bool) Option {
	return func(o *Options) {
		o.ReportCaller = report
	}
}

func UseReportTimestamp(report bool) Option {
	return func(o *Options) {
		o.ReportTimestamp = report
	}
}

func UseTimeFormat(format string) Option {
	return func(o *Options) {
		o.TimeFormat = format
	}
}

func UseStyles(s *Styles) Option {
	return func(o *Options) {
		o.Styles = s
	}
}

func AsDefault() Option {
	return func(o *Options) {
		o.Default = true
	}
}
