package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/utils/env"
	"github.com/docker/go-units"
)

// RotateWriter provides a thread-safe file writer with log rotation capabilities
// It manages log file size and maintains a maximum number of backup log files
type RotateWriter struct {
	// mu is a read-write mutex to ensure thread-safe file operations
	// It allows multiple concurrent reads but exclusive write access
	mu       sync.RWMutex
	file     *os.File
	size     int64
	maxSize  int64
	maxFiles int
	path     string
}

// NewRotateWriter creates a new RotateWriter with the specified logging configuration
// It initializes the writer with max file size and max number of backup files
func NewRotateWriter(cfg *config.LoggingConfig) (*RotateWriter, error) {
	// Convert human-readable size to bytes
	maxSize, err := units.FromHumanSize(cfg.Rotation.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max size format: %w", err)
	}

	w := &RotateWriter{
		maxSize:  maxSize,
		maxFiles: cfg.Rotation.MaxFiles,
		path:     env.GOMI_LOG_PATH,
	}

	// Open the initial log file
	if err := w.openFile(); err != nil {
		return nil, err
	}

	return w, nil
}

// Write implements the io.Writer interface
// It writes data to the log file and handles log rotation when the file size exceeds the maximum
func (w *RotateWriter) Write(p []byte) (n int, err error) {
	// Acquire an exclusive lock to ensure thread-safe write operations
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if writing this data would exceed the maximum file size
	writeLen := int64(len(p))
	if w.size+writeLen > w.maxSize {
		// Rotate the log file if size limit is reached
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// Write the data and update the current file size
	n, err = w.file.Write(p)
	w.size += int64(n)
	return n, err
}

// Close closes the current log file
func (w *RotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// openFile creates or opens the log file in append mode
// It ensures the log directory exists and sets the initial file size
func (w *RotateWriter) openFile() error {
	// Create the log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the log file with append and create flags
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Get file information to determine current size
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	// Close the previous file if it exists
	if w.file != nil {
		w.file.Close()
	}

	// Update the current file and its size
	w.file = f
	w.size = info.Size()
	return nil
}

// rotate handles log file rotation
// It creates a timestamped backup of the current log file and removes old backup files
func (w *RotateWriter) rotate() error {
	// Close the current file if it's open
	if w.file != nil {
		w.file.Close()
	}

	// Create a timestamped backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", w.path, timestamp)
	
	// Rename the current log file to the backup path
	if err := os.Rename(w.path, backupPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove old backup files if the number of backups exceeds the configured limit
	if err := w.removeOldFiles(); err != nil {
		return err
	}

	// Open a new log file
	return w.openFile()
}

// removeOldFiles removes excess backup log files
// It keeps only the most recent files up to the configured maximum
func (w *RotateWriter) removeOldFiles() error {
	// Skip if no file limit is set
	if w.maxFiles <= 0 {
		return nil
	}

	// Get the directory and base filename
	dir := filepath.Dir(w.path)
	base := filepath.Base(w.path)

	// Read all files in the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Collect backup log files
	var logFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), base+".") {
			logFiles = append(logFiles, f.Name())
		}
	}

	// Remove old backup files if the number exceeds the configured limit
	if len(logFiles) > w.maxFiles {
		sort.Strings(logFiles)
		for _, f := range logFiles[:len(logFiles)-w.maxFiles] {
			if err := os.Remove(filepath.Join(dir, f)); err != nil {
				return err
			}
		}
	}

	return nil
}
