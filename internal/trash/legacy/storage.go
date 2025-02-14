package legacy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/babarot/gomi/internal/fs"
	"github.com/babarot/gomi/internal/history"
	"github.com/babarot/gomi/internal/trash/core"
)

// Storage implements the core.Storage interface for legacy (.gomi) storage
type Storage struct {
	// Root directory for trash storage (~/.gomi)
	root string

	// History file path (~/.gomi/history.json)
	historyPath string

	// Configuration
	config *core.Config

	// In-memory cache of trash history
	// history *History
	history history.History
}

// Config holds legacy-specific configuration
type Config struct {
	// Root directory for trash storage
	TrashDir string

	// Whether to enable verbose output
	Verbose bool
}

// NewStorage creates a new legacy storage instance
func NewStorage(cfg *core.Config) (*Storage, error) {
	var root string
	if cfg.HomeTrashDir != "" {
		root = cfg.HomeTrashDir
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		root = filepath.Join(home, ".gomi")
	}

	s := &Storage{
		root:        root,
		historyPath: filepath.Join(root, "history.json"),
		config:      cfg,
		// history:     history.New(cfg.Core.TrashDir, cfg.History),
		history: history.New(cfg.TrashDir, cfg.History),
	}
	if err := s.history.Open(); err != nil {
		slog.Error("failed to open legacy history", "error", err)
	}

	// Create trash directory if it doesn't exist
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("failed to create trash directory: %w", err)
	}

	// // Load history
	// if err := s.loadHistory(); err != nil {
	// 	return nil, fmt.Errorf("failed to load history: %w", err)
	// }

	return s, nil
}

func (s *Storage) Info() *core.StorageInfo {
	return &core.StorageInfo{
		Location:  core.LocationHome,
		Root:      s.root,
		Available: true,
		Type:      core.StorageTypeLegacy,
	}
}

func (s *Storage) Put(src string) error {
	// Get absolute path
	abs, err := filepath.Abs(src)
	if err != nil {
		return core.NewStorageError("put", src, err)
	}

	// Generate unique ID for the file
	id := generateID()
	trashName := fmt.Sprintf("%s.%s", filepath.Base(abs), id)
	trashPath := filepath.Join(s.root, time.Now().Format("2006/01/02"), id, trashName)

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(trashPath), 0700); err != nil {
		return core.NewStorageError("put", src, err)
	}

	// Move file to trash
	if err := fs.MoveFile(abs, trashPath, false); err != nil {
		return core.NewStorageError("put", src, err)
	}

	// Add to history
	file := history.File{
		Name:      filepath.Base(abs),
		ID:        id,
		RunID:     id, // For compatibility with old format
		From:      abs,
		To:        trashPath,
		Timestamp: time.Now(),
	}
	s.history.Files = append(s.history.Files, file)

	// Save history
	if err := s.saveHistory(); err != nil {
		// Try to roll back the file move
		if moveErr := fs.MoveFile(trashPath, abs, false); moveErr != nil {
			return core.NewStorageError("put", src, fmt.Errorf("failed to save history and rollback failed: %v (original error: %w)", moveErr, err))
		}
		return core.NewStorageError("put", src, fmt.Errorf("failed to save history: %w", err))
	}

	return nil
}

func (s *Storage) Restore(file *core.File, dst string) error {
	if dst == "" {
		dst = file.OriginalPath
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return core.NewStorageError("restore", dst, err)
	}

	// Move file back
	if err := fs.MoveFile(file.TrashPath, dst, false); err != nil {
		return core.NewStorageError("restore", dst, err)
	}

	// Remove from history
	// s.history.Remove(file) TODO:

	// Save history
	if err := s.saveHistory(); err != nil {
		return core.NewStorageError("restore", dst, fmt.Errorf("failed to save history: %w", err))
	}

	return nil
}

func (s *Storage) Remove(file *core.File) error {
	// Remove the actual file
	if err := os.RemoveAll(file.TrashPath); err != nil {
		return core.NewStorageError("remove", file.TrashPath, err)
	}

	// Remove from history
	// s.history.Remove(filepath.Base(file.TrashPath))
	// s.history.Remove(file) TODO:

	// Save history
	if err := s.saveHistory(); err != nil {
		return core.NewStorageError("remove", file.TrashPath, fmt.Errorf("failed to save history: %w", err))
	}

	return nil
}

func (s *Storage) List() ([]*core.File, error) {
	var files []*core.File
	slog.Debug("storage.(legacy).List")

	for _, f := range s.history.Files {
		// Convert legacy File to core.File
		file := &core.File{
			Name:         f.Name,
			OriginalPath: f.From,
			TrashPath:    f.To,
			DeletedAt:    f.Timestamp,
		}

		// Get additional file info
		if info, err := os.Stat(f.To); err == nil {
			file.Size = info.Size()
			file.IsDir = info.IsDir()
			file.FileMode = info.Mode()
		}

		// slog.Debug("storage.List", "file", file)
		file.SetStorage(s)
		files = append(files, file)
	}

	return files, nil
}

// func (s *Storage) loadHistory() error {
// 	s.history = NewHistory()
//
// 	if _, err := os.Stat(s.historyPath); os.IsNotExist(err) {
// 		// No history file yet, start with empty history
// 		return nil
// 	}
//
// 	f, err := os.Open(s.historyPath)
// 	if err != nil {
// 		return fmt.Errorf("failed to open history file: %w", err)
// 	}
// 	defer f.Close()
//
// 	if err := json.NewDecoder(f).Decode(s.history); err != nil {
// 		return fmt.Errorf("failed to decode history file: %w", err)
// 	}
//
// 	return nil
// }

func (s *Storage) saveHistory() error {
	// Create history file atomically using a temporary file
	tmp, err := os.CreateTemp(filepath.Dir(s.historyPath), ".history.*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmp.Name()

	cleanup := func() {
		tmp.Close()
		os.Remove(tmpPath)
	}

	// Write history to temporary file
	if err := json.NewEncoder(tmp).Encode(s.history); err != nil {
		cleanup()
		return fmt.Errorf("failed to encode history: %w", err)
	}

	// Ensure data is written to disk
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("failed to sync history file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Rename temporary file to actual history file
	if err := os.Rename(tmpPath, s.historyPath); err != nil {
		cleanup()
		return fmt.Errorf("failed to save history file: %w", err)
	}

	return nil
}

// generateID generates a unique ID for a file
// This should match the format of the existing .gomi implementation
func generateID() string {
	return time.Now().Format("20060102150405")
}
