package debug

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShowExistingLogs(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := showExistingLogs(&buf, logPath)
	if err != nil {
		t.Fatalf("showExistingLogs() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "line1") {
		t.Errorf("output missing line1, got %q", output)
	}
	if !strings.Contains(output, "line3") {
		t.Errorf("output missing line3, got %q", output)
	}
}

func TestShowExistingLogs_NonExistent(t *testing.T) {
	var buf bytes.Buffer
	err := showExistingLogs(&buf, "/nonexistent/debug.log")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
	if !errors.Is(err, ErrLogFileNotFound) {
		t.Errorf("error = %v, want ErrLogFileNotFound", err)
	}
}

func TestShowExistingLogs_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "empty.log")
	if err := os.WriteFile(logPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := showExistingLogs(&buf, logPath)
	if err != nil {
		t.Fatalf("showExistingLogs() error = %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestLogs_FullMode(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")
	if err := os.WriteFile(logPath, []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err := Logs(&buf, logPath, false)
	if err != nil {
		t.Fatalf("Logs(live=false) error = %v", err)
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Errorf("output missing 'hello', got %q", buf.String())
	}
}

func TestLogs_NonExistent(t *testing.T) {
	var buf bytes.Buffer
	err := Logs(&buf, "/nonexistent/path.log", false)
	if !errors.Is(err, ErrLogFileNotFound) {
		t.Errorf("error = %v, want ErrLogFileNotFound", err)
	}
}

func TestConstants(t *testing.T) {
	if LiveMode != "live" {
		t.Errorf("LiveMode = %q, want %q", LiveMode, "live")
	}
	if FullMode != "full" {
		t.Errorf("FullMode = %q, want %q", FullMode, "full")
	}
}
