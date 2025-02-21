package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	charmlog "github.com/charmbracelet/log"
)

// Options represents logger configuration options
type Options struct {
	charmlog.Options // embed Options instead of pointer
	Writer           io.Writer
	Styles           *Styles
	Default          bool
	OutputFunc       func() (io.Writer, error)

	// Log rotation settings
	RotationEnabled  bool
	RotationMaxSize  string
	RotationMaxFiles int
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

	// If rotation is enabled and path is set, create rotate writer
	if o.RotationEnabled && o.OutputFunc != nil {
		if writer, err := o.createRotateWriter(); err == nil {
			o.Writer = writer
		}
	}
}

// createRotateWriter creates a new rotate writer based on current options
func (o *Options) createRotateWriter() (io.Writer, error) {
	w, err := o.OutputFunc()
	if err != nil {
		return nil, err
	}

	file, ok := w.(*os.File)
	if !ok {
		return nil, fmt.Errorf("output must be a file for rotation")
	}

	return newRotateWriter(file.Name(), o.RotationMaxSize, o.RotationMaxFiles)
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
		// If the error location's directory does not exist, create it
		if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return nil, fmt.Errorf("failed to create log file's directory: %w", err)
			}
		}
		return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	})
}

// EnableRotation enables log rotation with the specified settings
func EnableRotation(maxSize string, maxFiles int) Option {
	return func(o *Options) {
		o.RotationEnabled = true
		o.RotationMaxSize = maxSize
		o.RotationMaxFiles = maxFiles
	}
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
