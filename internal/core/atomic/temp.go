package atomic

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// TempFile represents a temporary file with cleanup capability
type TempFile struct {
	Path     string    // Path to the temporary file
	Created  time.Time // Creation time
	CleanErr error     // Error that occurred during cleanup if any
}

// TempManager handles temporary file operations
type TempManager struct {
	baseDir string // Base directory for temporary files
}

// NewTempManager creates a new TempManager instance
func NewTempManager(baseDir string) (*TempManager, error) {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}

	return &TempManager{
		baseDir: baseDir,
	}, nil
}

// CreateTemp creates a new temporary file
func (m *TempManager) CreateTemp(prefix string) (*TempFile, error) {
	id := uuid.New().String()
	tempPath := filepath.Join(
		m.baseDir,
		fmt.Sprintf("%s.%s.tmp", prefix, id),
	)

	// Create the file to reserve the name
	f, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	f.Close()

	return &TempFile{
		Path:    tempPath,
		Created: time.Now(),
	}, nil
}

// Cleanup removes the temporary file if it exists
func (f *TempFile) Cleanup() {
	if _, err := os.Stat(f.Path); err == nil {
		if err := os.Remove(f.Path); err != nil {
			f.CleanErr = NewCleanupError(f.Path, err)
		}
	}
}

// CleanupAll removes all temporary files older than the specified duration
func (m *TempManager) CleanupAll(olderThan time.Duration) error {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return fmt.Errorf("read temp directory: %w", err)
	}

	var errs []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > olderThan {
			path := filepath.Join(m.baseDir, entry.Name())
			if err := os.Remove(path); err != nil {
				errs = append(errs, NewCleanupError(path, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors occurred: %v", errs)
	}
	return nil
}

// GetPath returns the path to the temporary file
func (f *TempFile) GetPath() string {
	return f.Path
}

// SafeWriter wraps a temporary file with safe writing capabilities
type SafeWriter struct {
	temp     *TempFile
	file     *os.File
	finished bool
}

// NewSafeWriter creates a new SafeWriter
func (m *TempManager) NewSafeWriter(prefix string) (*SafeWriter, error) {
	temp, err := m.CreateTemp(prefix)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(temp.Path, os.O_WRONLY, 0600)
	if err != nil {
		temp.Cleanup()
		return nil, fmt.Errorf("open temp file: %w", err)
	}

	return &SafeWriter{
		temp: temp,
		file: file,
	}, nil
}

// Write writes data to the temporary file
func (w *SafeWriter) Write(p []byte) (n int, err error) {
	if w.finished {
		return 0, fmt.Errorf("write to finished writer")
	}
	return w.file.Write(p)
}

// Commit finalizes the write and moves the file to its destination
func (w *SafeWriter) Commit(dst string) error {
	if w.finished {
		return fmt.Errorf("commit finished writer")
	}
	w.finished = true

	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}
	w.file.Close()

	if err := os.Rename(w.temp.Path, dst); err != nil {
		return fmt.Errorf("rename to destination: %w", err)
	}

	return nil
}

// Cleanup removes the temporary file
func (w *SafeWriter) Cleanup() {
	w.file.Close()
	w.temp.Cleanup()
}
