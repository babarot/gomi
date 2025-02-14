// internal/trash/manager.go
package trash

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/babarot/gomi/internal/trash/core"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

type Strategy string

const (
	TrashStrategyXDG       Strategy = "xdg"
	TrashStrategyLegacy    Strategy = "legacy"
	TrashStrategyComposite Strategy = "composite"
)

// Manager handles multiple trash storage implementations
type Manager struct {
	storages []core.Storage
	config   *core.Config
	strategy Strategy
}

// NewManager creates a new trash manager with the given configuration
func NewManager(cfg *core.Config) (*Manager, error) {

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	var storages []core.Storage

	// Initialize primary storage
	primaryStorage, err := NewStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize primary storage: %w", err)
	}
	slog.Info("primaryStorage set", "storage", primaryStorage.Info().Type)
	storages = append(storages, primaryStorage)

	// Initialize fallback storage if enabled
	if cfg.EnableHomeFallback && cfg.Type == core.StorageTypeXDG {
		fallbackCfg := *cfg // Create a copy of the config
		fallbackCfg.Type = core.StorageTypeLegacy
		fallbackCfg.UseXDG = false
		fallbackStorage, err := NewStorage(&fallbackCfg)
		if err != nil {
			slog.Warn("failed to initialize fallback storage", "error", err)
		} else {
			storages = append(storages, fallbackStorage)
		}
	}

	if legacy, _ := DetectExistingLegacy(); legacy && cfg.Type == core.StorageTypeXDG {
		slog.Debug("found legacy storage in XDG enabled")
		ls, err := newLegacyStorage(cfg)
		if err != nil {
			slog.Error("failed to set legacy storage", "error", err)
		}
		storages = append(storages, ls)
	}

	if len(storages) == 0 {
		return nil, errors.New("no storage backend configured")
	}

	// var sts []core.StorageType
	// for _, storage := range storages {
	// 	switch storage.Info().Type {
	// 		case
	// 	}
	// }
	sts := lo.Map(storages, func(s core.Storage, index int) core.StorageType {
		return s.Info().Type
	})

	var strategy Strategy
	switch len(slices.Compact(sts)) {
	case 1:
		switch sts[0] {
		case core.StorageTypeXDG:
			strategy = TrashStrategyXDG
		case core.StorageTypeLegacy:
			strategy = TrashStrategyLegacy
		}
	case 2:
		strategy = TrashStrategyComposite
	}

	slog.Info("Trash Strategy!", "strategy", strategy)
	return &Manager{
		storages: storages,
		config:   cfg,
		strategy: strategy,
	}, nil
}

// Put moves the file at src path to trash
func (m *Manager) Put(src string) error {
	absPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	fi, err := os.Lstat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Try each storage in order until one succeeds
	var lastErr error
	for _, storage := range m.storages {
		err := storage.Put(absPath)
		if err == nil {
			if m.config.Verbose {
				if fi.IsDir() {
					fmt.Printf("Moved directory %s to trash\n", absPath)
				} else {
					fmt.Printf("Moved file %s to trash\n", absPath)
				}
			}
			return nil
		}
		lastErr = err
		slog.Debug("storage failed to put file", "storage", storage.Info().Root, "error", err)
	}

	return fmt.Errorf("all storage backends failed to put file: %w", lastErr)
}

// List returns all files from all storages
func (m *Manager) List() ([]*core.File, error) {
	var allFiles []*core.File
	var errs []error

	for _, storage := range m.storages {
		files, err := storage.List()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to list files from %s: %w", storage.Info().Root, err))
			continue
		}
		allFiles = append(allFiles, files...)
	}

	if len(allFiles) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all storage backends failed: %v", errs)
	}

	return allFiles, nil
}

// Restore restores the given file
func (m *Manager) Restore(file *core.File, dst string) error {
	// Find the appropriate storage for this file
	var targetStorage core.Storage
	for _, storage := range m.storages {
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
	// if _, err := os.Stat(dst); err == nil && !m.config.Force { TODO:
	if _, err := os.Stat(dst); err == nil {
		return core.ErrFileExists
	}

	return targetStorage.Restore(file, dst)
}

// Remove permanently removes the file from trash
func (m *Manager) Remove(file *core.File) error {
	// Find the appropriate storage for this file
	var targetStorage core.Storage
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
func (m *Manager) ListStorages() []*core.StorageInfo {
	var infos []*core.StorageInfo
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
