package legacy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/legacy/history"
	"github.com/babarot/gomi/internal/utils/fs"
)

// Storage implements the trash.Storage interface for legacy (.gomi) storage
type Storage struct {
	// mu protects history access for concurrent Put/Restore/Remove calls
	mu sync.Mutex

	// Root directory for trash storage (~/.gomi)
	root string

	// Configuration
	config trash.Config

	// History file path (~/.gomi/history.json)
	historyPath string

	// In-memory cache of trash history
	history history.History
}

// NewStorage creates a new legacy storage instance
func NewStorage(cfg trash.Config) (trash.Storage, error) {
	slog.Info("initialize legacy storage")

	var root string
	if cfg.GomiDir != "" {
		root = cfg.GomiDir
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		root = filepath.Join(home, ".gomi")
	}

	s := &Storage{
		root:        root,
		config:      cfg,
		historyPath: filepath.Join(root, history.Filename),
		history:     history.New(cfg.GomiDir, cfg.History),
	}
	slog.Debug("legacy storage",
		"gomiDir", cfg.GomiDir,
		"historyPath", s.historyPath)

	// Create trash directory if it doesn't exist
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, fmt.Errorf("failed to create trash directory: %w", err)
	}

	// Load history
	if err := s.loadHistory(); err != nil {
		return nil, fmt.Errorf("failed to load history: %w", err)
	}

	return s, nil
}

func (s *Storage) Info() *trash.StorageInfo {
	return &trash.StorageInfo{
		Location:  trash.LocationHome,
		Trashes:   []string{s.root},
		Available: true,
		Type:      trash.StorageTypeLegacy,
	}
}

func (s *Storage) Put(src string) error {
	// Get absolute path
	abs, err := filepath.Abs(src)
	if err != nil {
		return trash.NewStorageError("put", src, err)
	}

	id := uuid.New().String()
	trashName := fmt.Sprintf("%s.%s", filepath.Base(abs), id)
	trashPath := filepath.Join(s.root, time.Now().Format("2006/01/02"), id, trashName)

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(trashPath), 0700); err != nil {
		return trash.NewStorageError("put", src, err)
	}

	// Move file to trash (with fallback copy for cross-device moves)
	if err := fs.Move(abs, trashPath, true); err != nil {
		return trash.NewStorageError("put", src, err)
	}

	// Add to history (protected by mutex for concurrent Put calls)
	s.mu.Lock()
	s.history.Add(history.File{
		Name:      filepath.Base(abs),
		ID:        id,
		RunID:     id, // For compatibility with old format
		From:      abs,
		To:        trashPath,
		Timestamp: time.Now(),
	})

	// Save history
	if err := s.saveHistory(); err != nil {
		s.mu.Unlock()
		// Try to roll back the file move
		if moveErr := fs.Move(trashPath, abs, true); moveErr != nil {
			return trash.NewStorageError(
				"put",
				src,
				fmt.Errorf("failed to save history and rollback failed: %w (original error: %w)", moveErr, err))
		}
		return trash.NewStorageError(
			"put",
			src,
			fmt.Errorf("failed to save history: %w", err))
	}
	s.mu.Unlock()

	return nil
}

func (s *Storage) List() ([]*trash.File, error) {
	s.mu.Lock()
	filtered := s.history.Filter()
	s.mu.Unlock()

	var files []*trash.File

	for _, f := range filtered {
		// Convert legacy File to trash.File
		file := &trash.File{
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

		file.SetStorage(s)
		files = append(files, file)
	}

	return files, nil
}

func (s *Storage) Restore(file *trash.File, dst string) error {
	if dst == "" {
		dst = file.OriginalPath
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return trash.NewStorageError("restore", dst, err)
	}

	// Move file back (with fallback copy for cross-device moves)
	if err := fs.Move(file.TrashPath, dst, true); err != nil {
		return trash.NewStorageError("restore", dst, err)
	}

	// Remove from history (protected by mutex)
	s.mu.Lock()
	s.history.RemoveByPath(file.TrashPath)
	if err := s.saveHistory(); err != nil {
		s.mu.Unlock()
		return trash.NewStorageError("restore", dst, fmt.Errorf("failed to save history: %w", err))
	}
	s.mu.Unlock()

	return nil
}

func (s *Storage) Remove(file *trash.File) error {
	// Remove the actual file
	if err := os.RemoveAll(file.TrashPath); err != nil {
		return trash.NewStorageError("remove", file.TrashPath, err)
	}

	// Remove from history (protected by mutex)
	s.mu.Lock()
	s.history.RemoveByPath(file.TrashPath)
	if err := s.saveHistory(); err != nil {
		s.mu.Unlock()
		return trash.NewStorageError("remove", file.TrashPath, fmt.Errorf("failed to save history: %w", err))
	}
	s.mu.Unlock()

	return nil
}

func (s *Storage) loadHistory() error {
	if err := s.history.Open(); err != nil {
		slog.Error("failed to open legacy history", "error", err)
		return err
	}

	/* TODO: Remove this logic or keep instead of s.history.Open()

	if _, err := os.Stat(s.historyPath); os.IsNotExist(err) {
		// No history file yet, start with empty history
		return nil
	}

	f, err := os.Open(s.historyPath)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&s.history); err != nil {
		return fmt.Errorf("failed to decode history file: %w", err)
	}
	*/

	return nil
}

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
