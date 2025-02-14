package trash

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/trash/legacy"
	"github.com/babarot/gomi/internal/trash/xdg"
)

// StorageType represents the type of trash storage
type StorageType int

const (
	// StorageTypeXDG represents XDG-compliant trash storage
	StorageTypeXDG StorageType = iota

	// StorageTypeLegacy represents legacy (.gomi) trash storage
	StorageTypeLegacy
)

// StorageConfig holds configuration for creating Storage instances
type StorageConfig struct {
	// Type determines which storage implementation to use
	Type StorageType

	// HomeTrashDir specifies a custom home trash directory
	// For XDG: defaults to ~/.local/share/Trash
	// For Legacy: defaults to ~/.gomi
	HomeTrashDir string

	// EnableHomeFallback enables fallback to home trash when external trash fails
	EnableHomeFallback bool

	// Verbose enables detailed logging
	Verbose bool
}

// NewStorage creates a new Storage instance based on the provided configuration
func NewStorage(cfg *StorageConfig) (Storage, error) {
	switch cfg.Type {
	case StorageTypeXDG:
		return newXDGStorage(cfg)
	case StorageTypeLegacy:
		return newLegacyStorage(cfg)
	default:
		return nil, fmt.Errorf("unknown storage type: %v", cfg.Type)
	}
}

// newXDGStorage creates a new XDG-compliant trash storage
func newXDGStorage(cfg *StorageConfig) (Storage, error) {
	// Determine home trash directory
	homeTrash := cfg.HomeTrashDir
	if homeTrash == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		homeTrash = filepath.Join(home, ".local", "share", "Trash")
	}

	// Create an XDG storage instance
	storage, err := xdg.NewStorage(&xdg.Config{
		HomeTrashDir:       homeTrash,
		EnableHomeFallback: cfg.EnableHomeFallback,
		Verbose:            cfg.Verbose,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create XDG storage: %w", err)
	}

	return storage, nil
}

// newLegacyStorage creates a new legacy (.gomi) trash storage
func newLegacyStorage(cfg *StorageConfig) (Storage, error) {
	// Determine home trash directory
	homeTrash := cfg.HomeTrashDir
	if homeTrash == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		homeTrash = filepath.Join(home, ".gomi")
	}

	// Create a legacy storage instance
	storage, err := legacy.NewStorage(&legacy.Config{
		TrashDir: homeTrash,
		Verbose:  cfg.Verbose,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create legacy storage: %w", err)
	}

	return storage, nil
}

func DetectLegacy() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for legacy storage
	legacyPath := filepath.Join(home, ".gomi")
	if fi, err := os.Stat(legacyPath); err == nil && fi.IsDir() {
		return true, nil
	}

	// Default to XDG storage
	return false, nil
}

// DetectExistingStorage detects what type of storage is in use
// by checking for existing trash directories
func DetectExistingStorage() (StorageType, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return StorageTypeXDG, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check for legacy storage
	legacyPath := filepath.Join(home, ".gomi")
	if fi, err := os.Stat(legacyPath); err == nil && fi.IsDir() {
		return StorageTypeLegacy, nil
	}

	// Default to XDG storage
	return StorageTypeXDG, nil
}

// ValidateStorageConfig validates the storage configuration
func ValidateStorageConfig(cfg *StorageConfig) error {
	if cfg.HomeTrashDir != "" {
		if !filepath.IsAbs(cfg.HomeTrashDir) {
			return fmt.Errorf("home trash directory must be an absolute path: %s", cfg.HomeTrashDir)
		}
	}

	return nil
}
