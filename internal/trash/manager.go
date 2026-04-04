package trash

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/babarot/gomi/internal/utils/log"
)

// Strategy represents the trash management strategy
type Strategy string

const (
	// StrategyXDG uses XDG trash specification
	StrategyXDG Strategy = "xdg"

	// StrategyLegacy uses legacy (.gomi) format
	StrategyLegacy Strategy = "legacy"

	// StrategyAuto uses multiple storage backends
	StrategyAuto Strategy = "auto"

	// StrategyNone disables trash functionality, preventing files from being moved to trash
	StrategyNone Strategy = ""
)

// Trash defines the interface for trash operations.
// It is implemented by Manager and can be mocked in tests.
type Trash interface {
	Put(src string) error
	List() ([]*File, error)
	Restore(file *File, dst string) error
	Remove(file *File) error
}

// Manager handles multiple trash storage implementations
type Manager struct {
	storages []Storage
	config   Config
	strategy Strategy
}

// ManagerOption is a function type for configuring Manager
type ManagerOption func(*Manager)

// WithStorage adds a new storage implementation to the manager
func WithStorage(constructor StorageConstructor) ManagerOption {
	return func(m *Manager) {
		storage, err := constructor(m.config)
		if err != nil {
			slog.Error("failed to initialize storage", "error", err)
			return
		}
		m.storages = append(m.storages, storage)
	}
}

// NewManager creates a new trash manager with the given configuration and options
func NewManager(cfg Config, opts ...ManagerOption) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	m := &Manager{
		config:   cfg,
		storages: make([]Storage, 0),
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(m)
	}

	if len(m.storages) == 0 {
		return nil, errors.New("no storage backend configured")
	}

	// Determine trash strategy automatically if not set
	if m.strategy == StrategyNone {
		slog.Debug("determine strategy based on current storages")
		m.strategy = determineStrategy(m.storages)
	}
	slog.Info(log.Highlight("trash manager"), "strategy", m.strategy)

	return m, nil
}

// determineStrategy determines the strategy based on available storages
func determineStrategy(storages []Storage) Strategy {
	if len(storages) == 0 {
		return StrategyNone
	}

	if len(storages) == 1 {
		switch storages[0].Info().Type {
		case StorageTypeXDG:
			return StrategyXDG
		case StorageTypeLegacy:
			return StrategyLegacy
		}
	}

	return StrategyAuto
}

// Put moves the file at src path to trash
func (m *Manager) Put(src string) error {
	path, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	slog.Debug("putting file to trash",
		"source", src,
		"absolute_path", path,
		"strategy", m.strategy,
		"storages", len(m.storages))

	fi, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	var lastErr error
	for _, storage := range m.storages {
		err := storage.Put(path)
		if err == nil {
			if fi.IsDir() {
				slog.Debug("moved directory to trash", "path", path)
			} else {
				slog.Debug("moved file to trash", "path", path)
			}
			return nil
		}
		lastErr = err
		slog.Debug("storage failed to put file",
			"trashes", storage.Info().Trashes,
			"error", err)
	}

	return fmt.Errorf("all storage backends failed to put file: %w", lastErr)
}

// List returns all files from all storage backends
func (m *Manager) List() ([]*File, error) {
	slog.Debug("storage manager list")

	type result struct {
		files []*File
		err   error
	}

	ch := make(chan result, len(m.storages))
	for _, storage := range m.storages {
		go func(s Storage) {
			files, err := s.List()
			if err != nil {
				ch <- result{err: fmt.Errorf("failed to list files from %s: %w",
					s.Info().Trashes, err)}
				return
			}
			slog.Info("list files",
				"storage_type", s.Info().Type,
				"len(files)", len(files))
			ch <- result{files: files}
		}(storage)
	}

	var allFiles []*File
	var errs []error
	for range m.storages {
		r := <-ch
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		allFiles = append(allFiles, r.files...)
	}

	if len(allFiles) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all storage backends failed: %v", errs)
	}

	return allFiles, nil
}

// Restore restores the given file
func (m *Manager) Restore(file *File, dst string) error {
	storage, err := m.findStorageForFile(file)
	if err != nil {
		return err
	}

	if dst == "" {
		dst = file.OriginalPath
	}

	// Check if destination exists
	if _, err := os.Stat(dst); err == nil {
		return ErrFileExists
	}

	return storage.Restore(file, dst)
}

// Remove permanently removes the file from trash
func (m *Manager) Remove(file *File) error {
	storage, err := m.findStorageForFile(file)
	if err != nil {
		return err
	}

	return storage.Remove(file)
}

// findStorageForFile returns the storage backend that manages the given file,
// determined by matching the file's trash path against each storage's root paths.
func (m *Manager) findStorageForFile(file *File) (Storage, error) {
	for _, storage := range m.storages {
		for _, trashRoot := range storage.Info().Trashes {
			if strings.HasPrefix(file.TrashPath, trashRoot) {
				return storage, nil
			}
		}
	}
	return nil, errors.New("file does not belong to any known storage")
}

// ListStorages returns information about all available storage backends
func (m *Manager) ListStorages() []*StorageInfo {
	var infos []*StorageInfo
	for _, storage := range m.storages {
		infos = append(infos, storage.Info())
	}
	return infos
}

// IsPrimaryStorageAvailable checks if the primary storage is available
func (m *Manager) IsPrimaryStorageAvailable() bool {
	if len(m.storages) == 0 {
		return false
	}
	return m.storages[0].Info().Available
}
