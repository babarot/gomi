package log

import (
	"io"
	"testing"
)

func TestNew(t *testing.T) {
	Reset()
	defer Reset()

	logger, err := New(
		UseLevel(DebugLevel),
		UseOutput(io.Discard),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestNew_AsDefault(t *testing.T) {
	Reset()
	defer Reset()

	_, err := New(
		UseLevel(InfoLevel),
		UseOutput(io.Discard),
		AsDefault(),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// After AsDefault, Default() should return a non-nil logger
	def := Default()
	if def == nil {
		t.Error("Default() should return non-nil after AsDefault()")
	}
}

func TestNew_WithOptions(t *testing.T) {
	Reset()
	defer Reset()

	// Test New() with multiple options combined.
	// Uses io.Discard to avoid file handle leaks on Windows.
	logger, err := New(
		UseOutput(io.Discard),
		UseLevel(DebugLevel),
		UseReportCaller(true),
		UseReportTimestamp(true),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
}

func TestDefault(t *testing.T) {
	Reset()
	defer Reset()

	d := Default()
	if d == nil {
		t.Fatal("Default() returned nil")
	}
}

func TestReset(t *testing.T) {
	Reset()

	// After reset, Default() should create a new logger
	d := Default()
	if d == nil {
		t.Fatal("Default() returned nil after Reset()")
	}
}
