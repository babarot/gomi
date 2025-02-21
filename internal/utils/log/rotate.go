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

type RotateWriter struct {
	mu       sync.Mutex
	file     *os.File
	size     int64
	maxSize  int64
	maxFiles int
	path     string
}

func NewRotateWriter(cfg *config.LoggingConfig) (*RotateWriter, error) {
	maxSize, err := units.FromHumanSize(cfg.Rotation.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max size format: %w", err)
	}

	w := &RotateWriter{
		maxSize:  maxSize,
		maxFiles: cfg.Rotation.MaxFiles,
		path:     env.GOMI_LOG_PATH,
	}

	if err := w.openFile(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *RotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()

	writeLen := int64(len(p))
	if w.size+writeLen > w.maxSize {
		w.mu.Unlock()
		if err := w.rotate(); err != nil {
			return 0, err
		}
		w.mu.Lock()
	}

	n, err = w.file.Write(p)
	w.size += int64(n)
	w.mu.Unlock()
	return n, err
}

func (w *RotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *RotateWriter) openFile() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	if w.file != nil {
		w.file.Close()
	}

	w.file = f
	w.size = info.Size()
	return nil
}

func (w *RotateWriter) rotate() error {
	if w.file != nil {
		w.file.Close()
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", w.path, timestamp)
	if err := os.Rename(w.path, backupPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := w.removeOldFiles(); err != nil {
		return err
	}

	return w.openFile()
}

func (w *RotateWriter) removeOldFiles() error {
	if w.maxFiles <= 0 {
		return nil
	}

	dir := filepath.Dir(w.path)
	base := filepath.Base(w.path)

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var logFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), base+".") {
			logFiles = append(logFiles, f.Name())
		}
	}

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
