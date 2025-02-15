package trash

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	slog.Info("trash manager", "strategy", m.strategy)

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
			"storage", storage.Info().Root,
			"error", err)
	}

	return fmt.Errorf("all storage backends failed to put file: %w", lastErr)
}

// List returns all files from all storage backends
func (m *Manager) List() ([]*File, error) {
	var allFiles []*File
	var errs []error

	slog.Debug("storage manager list")

	for _, storage := range m.storages {
		files, err := storage.List()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to list files from %s: %w",
				storage.Info().Root, err))
			continue
		}
		allFiles = append(allFiles, files...)
		slog.Debug("list files",
			"storage_type",
			storage.Info().Type,
			"len(files)",
			len(files),
		)
	}

	if len(allFiles) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all storage backends failed: %v", errs)
	}

	return allFiles, nil
}

// Restore restores the given file
func (m *Manager) Restore(file *File, dst string) error {
	// Find the appropriate storage for this file
	var targetStorage Storage
	for _, storage := range m.storages {
		slog.Debug("checking storage",
			"type", storage.Info().Type,
			"file", file.Name,
			"info", storage.Info())
		if strings.HasPrefix(file.TrashPath, storage.Info().Root) {
			targetStorage = storage
			break
		}
	}

	if targetStorage == nil {
		return errors.New("file does not belong to any known storage")
	}

	if dst == "" {
		dst = file.OriginalPath
	}

	// Check if destination exists
	if _, err := os.Stat(dst); err == nil {
		return ErrFileExists
	}

	return targetStorage.Restore(file, dst)
}

// Remove permanently removes the file from trash
func (m *Manager) Remove(file *File) error {
	// Find the appropriate storage for this file
	var targetStorage Storage
	for _, storage := range m.storages {
		if strings.HasPrefix(file.TrashPath, storage.Info().Root) {
			targetStorage = storage
			break
		}
	}

	if targetStorage == nil {
		return errors.New("file does not belong to any known storage")
	}

	return targetStorage.Remove(file)
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
