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

func TestNew_WithRotation(t *testing.T) {
	Reset()
	defer Reset()

	dir := t.TempDir()
	logPath := dir + "/test.log"

	logger, err := New(
		UseOutputPath(logPath),
		EnableRotation("1MB", 3),
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
