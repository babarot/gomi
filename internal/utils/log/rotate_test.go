package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRotateWriter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	rw, err := newRotateWriter(path, "1MB", 3)
	if err != nil {
		t.Fatalf("newRotateWriter() error = %v", err)
	}
	defer rw.Close()

	// docker/go-units uses SI units: 1MB = 1,000,000
	if rw.maxSize != 1000000 {
		t.Errorf("maxSize = %d, want %d", rw.maxSize, 1000000)
	}
	if rw.maxFiles != 3 {
		t.Errorf("maxFiles = %d, want 3", rw.maxFiles)
	}

	// File should have been created
	if _, err := os.Stat(path); err != nil {
		t.Errorf("log file not created: %v", err)
	}
}

func TestNewRotateWriter_InvalidSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	_, err := newRotateWriter(path, "notasize", 3)
	if err == nil {
		t.Fatal("expected error for invalid size")
	}
}

func TestRotateWriter_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	rw, err := newRotateWriter(path, "1KB", 3)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()

	// Write some data
	data := []byte("hello world\n")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}
	if rw.size != int64(len(data)) {
		t.Errorf("size = %d, want %d", rw.size, len(data))
	}
}

func TestRotateWriter_Rotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Use very small max size to trigger rotation
	rw, err := newRotateWriter(path, "50B", 2)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()

	// Write enough data to trigger rotation
	bigData := []byte(strings.Repeat("x", 60))
	if _, err := rw.Write(bigData); err != nil {
		t.Fatalf("first Write() error = %v", err)
	}

	// This write should trigger rotation
	if _, err := rw.Write(bigData); err != nil {
		t.Fatalf("second Write() error = %v", err)
	}

	// Check that backup files were created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var logFiles int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.log") {
			logFiles++
		}
	}
	// Should have at least the current file + 1 backup
	if logFiles < 2 {
		t.Errorf("expected at least 2 log files, got %d", logFiles)
	}
}

func TestRotateWriter_RemoveOldFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Create some fake backup files
	for _, suffix := range []string{".20240101-120000", ".20240102-120000", ".20240103-120000", ".20240104-120000"} {
		os.WriteFile(path+suffix, []byte("old"), 0644)
	}

	rw, err := newRotateWriter(path, "1KB", 2)
	if err != nil {
		t.Fatal(err)
	}
	defer rw.Close()

	if err := rw.removeOldFiles(); err != nil {
		t.Fatal(err)
	}

	// Check remaining backups
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	var backups int
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.log.") {
			backups++
		}
	}
	if backups > 2 {
		t.Errorf("expected at most 2 backups, got %d", backups)
	}
}

func TestRotateWriter_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	rw, err := newRotateWriter(path, "1KB", 3)
	if err != nil {
		t.Fatal(err)
	}

	if err := rw.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Close on nil file should be safe
	rw.file = nil
	if err := rw.Close(); err != nil {
		t.Fatalf("Close() on nil file error = %v", err)
	}
}
