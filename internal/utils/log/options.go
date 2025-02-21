package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	charmlog "github.com/charmbracelet/log"
	"github.com/docker/go-units"
)

// Options represents logger configuration options
type Options struct {
	charmlog.Options // embed Options instead of pointer

	Writer     io.Writer
	Styles     *Styles
	Default    bool
	OutputFunc func() (io.Writer, error)

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
func (o *Options) Apply(opts ...Option) error {
	// Apply all options first
	for _, opt := range opts {
		opt(o)
	}

	// If rotation is enabled, validate and setup rotation
	if o.RotationEnabled {
		if err := o.setupRotation(); err != nil {
			return fmt.Errorf("failed to setup rotation: %w", err)
		}
	}

	return nil
}

func (o *Options) setupRotation() error {
	var w io.Writer
	var err error

	// Get the writer either from OutputFunc or from the already set Writer
	if o.OutputFunc != nil {
		w, err = o.OutputFunc()
		if err != nil {
			return fmt.Errorf("failed to get output writer: %w", err)
		}
	} else if o.Writer != nil {
		w = o.Writer
	} else {
		return fmt.Errorf("rotation enabled but no output specified")
	}

	// Check if the writer is a file
	file, ok := w.(*os.File)
	if !ok {
		return fmt.Errorf("rotation can only be enabled for file output")
	}

	// Create rotate writer
	rw, err := newRotateWriter(file.Name(), o.RotationMaxSize, o.RotationMaxFiles)
	if err != nil {
		return fmt.Errorf("failed to create rotate writer: %w", err)
	}

	o.Writer = rw
	return nil
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

// EnableRotation enables log rotation with the specified settings.
// If the settings are invalid (empty or zero), rotation will be silently disabled.
func EnableRotation(maxSize string, maxFiles int) Option {
	return func(o *Options) {
		// Skip rotation if maxSize is empty or invalid
		if maxSize == "" {
			return
		}
		if size, err := units.FromHumanSize(maxSize); err != nil || size <= 0 {
			return
		}

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
