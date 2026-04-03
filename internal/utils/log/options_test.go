package log

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	o := DefaultOptions()
	if o.Level != InfoLevel {
		t.Errorf("Level = %v, want InfoLevel", o.Level)
	}
	if o.Writer != os.Stderr {
		t.Error("Writer should default to os.Stderr")
	}
	if o.ReportCaller {
		t.Error("ReportCaller should default to false")
	}
	if o.ReportTimestamp {
		t.Error("ReportTimestamp should default to false")
	}
	if o.Styles == nil {
		t.Error("Styles should not be nil")
	}
}

func TestUseLevel(t *testing.T) {
	o := DefaultOptions()
	UseLevel(DebugLevel)(o)
	if o.Level != DebugLevel {
		t.Errorf("Level = %v, want DebugLevel", o.Level)
	}
}

func TestUseOutput(t *testing.T) {
	o := DefaultOptions()
	var buf bytes.Buffer
	UseOutput(&buf)(o)
	if o.Writer != &buf {
		t.Error("Writer not set correctly")
	}
}

func TestUseOutputPath(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "subdir", "test.log")

	o := DefaultOptions()
	UseOutputPath(logPath)(o)

	if o.OutputFunc == nil {
		t.Fatal("OutputFunc should be set")
	}

	w, err := o.OutputFunc()
	if err != nil {
		t.Fatalf("OutputFunc() error = %v", err)
	}
	if f, ok := w.(*os.File); ok {
		f.Close()
	}

	// Directory should have been created
	if _, err := os.Stat(filepath.Dir(logPath)); err != nil {
		t.Errorf("log directory not created: %v", err)
	}
}

func TestUseOutputPath_Empty(t *testing.T) {
	o := DefaultOptions()
	UseOutputPath("")(o)

	w, err := o.OutputFunc()
	if err != nil {
		t.Fatalf("OutputFunc() error = %v", err)
	}
	if w != os.Stderr {
		t.Error("empty path should return os.Stderr")
	}
}

func TestEnableRotation(t *testing.T) {
	t.Run("valid settings", func(t *testing.T) {
		o := DefaultOptions()
		EnableRotation("10MB", 3)(o)
		if !o.RotationEnabled {
			t.Error("RotationEnabled should be true")
		}
		if o.RotationMaxSize != "10MB" {
			t.Errorf("RotationMaxSize = %q, want %q", o.RotationMaxSize, "10MB")
		}
		if o.RotationMaxFiles != 3 {
			t.Errorf("RotationMaxFiles = %d, want 3", o.RotationMaxFiles)
		}
	})

	t.Run("empty maxSize disables rotation", func(t *testing.T) {
		o := DefaultOptions()
		EnableRotation("", 3)(o)
		if o.RotationEnabled {
			t.Error("RotationEnabled should be false for empty maxSize")
		}
	})

	t.Run("invalid maxSize disables rotation", func(t *testing.T) {
		o := DefaultOptions()
		EnableRotation("notasize", 3)(o)
		if o.RotationEnabled {
			t.Error("RotationEnabled should be false for invalid maxSize")
		}
	})
}

func TestUseReportCaller(t *testing.T) {
	o := DefaultOptions()
	UseReportCaller(true)(o)
	if !o.ReportCaller {
		t.Error("ReportCaller should be true")
	}
}

func TestUseReportTimestamp(t *testing.T) {
	o := DefaultOptions()
	UseReportTimestamp(true)(o)
	if !o.ReportTimestamp {
		t.Error("ReportTimestamp should be true")
	}
}

func TestUseTimeFormat(t *testing.T) {
	o := DefaultOptions()
	UseTimeFormat(time.Kitchen)(o)
	if o.TimeFormat != time.Kitchen {
		t.Errorf("TimeFormat = %q, want %q", o.TimeFormat, time.Kitchen)
	}
}

func TestAsDefault(t *testing.T) {
	o := DefaultOptions()
	AsDefault()(o)
	if !o.Default {
		t.Error("Default should be true")
	}
}

func TestApply(t *testing.T) {
	o := DefaultOptions()
	err := o.Apply(
		UseLevel(WarnLevel),
		UseReportCaller(true),
		UseOutput(io.Discard),
	)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if o.Level != WarnLevel {
		t.Errorf("Level = %v, want WarnLevel", o.Level)
	}
	if !o.ReportCaller {
		t.Error("ReportCaller should be true")
	}
}
