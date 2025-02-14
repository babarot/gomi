// internal/trash/factory.go
package trash

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/trash/core"
	"github.com/babarot/gomi/internal/trash/legacy"
	"github.com/babarot/gomi/internal/trash/xdg"
)

// NewStorage creates a new Storage instance based on the provided configuration
func NewStorage(cfg *core.Config) (core.Storage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	switch cfg.Type {
	case core.StorageTypeXDG:
		return newXDGStorage(cfg)
	case core.StorageTypeLegacy:
		return newLegacyStorage(cfg)
	default:
		return nil, fmt.Errorf("unknown storage type: %v", cfg.Type)
	}
}

// newXDGStorage creates a new XDG-compliant trash storage
func newXDGStorage(cfg *core.Config) (core.Storage, error) {
	if cfg.HomeTrashDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.HomeTrashDir = filepath.Join(home, ".local", "share", "Trash")
	}

	storage, err := xdg.NewStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create XDG storage: %w", err)
	}

	return storage, nil
}

// newLegacyStorage creates a new legacy (.gomi) trash storage
func newLegacyStorage(cfg *core.Config) (core.Storage, error) {
	if cfg.HomeTrashDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cfg.HomeTrashDir = filepath.Join(home, ".gomi")
	}

	storage, err := legacy.NewStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create legacy storage: %w", err)
	}

	return storage, nil
}

// DetectExistingStorage detects what type of storage is in use
// by checking for existing trash directories
func DetectExistingStorage() (core.StorageType, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return core.StorageTypeXDG, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for legacy storage
	legacyPath := filepath.Join(home, ".gomi")
	if fi, err := os.Stat(legacyPath); err == nil && fi.IsDir() {
		return core.StorageTypeLegacy, nil
	}

	// Check for XDG storage
	xdgPath := filepath.Join(home, ".local", "share", "Trash")
	if fi, err := os.Stat(xdgPath); err == nil && fi.IsDir() {
		return core.StorageTypeXDG, nil
	}

	// Default to XDG storage if no existing storage is found
	return core.StorageTypeXDG, nil
}

// AutoConfigure creates a configuration based on the existing environment
func AutoConfigure() (*core.Config, error) {
	storageType, err := DetectExistingStorage()
	if err != nil {
		return nil, err
	}

	cfg := core.NewDefaultConfig()
	cfg.Type = storageType
	cfg.UseXDG = (storageType == core.StorageTypeXDG)

	return cfg, nil
}

func DetectExistingLegacy() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for legacy storage
	legacyPath := filepath.Join(home, ".gomi")
	if fi, err := os.Stat(legacyPath); err == nil && fi.IsDir() {
		return true, nil
	}

	// Default to XDG storage if no existing storage is found
	return false, nil
}
