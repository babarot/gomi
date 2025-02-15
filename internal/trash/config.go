package trash

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/config"
)

// Config holds the unified configuration for trash management.
// It maintains a deliberate separation between user-facing strategy (Strategy)
// and internal implementation details (Type) to provide clear boundaries between
// configuration intent and actual storage implementation.
type Config struct {
	// Strategy determines which trash specification to use:
	// - "auto": automatically detect and use both XDG and legacy if available
	// - "xdg": strictly follow XDG trash specification
	// - "legacy": use gomi's legacy trash format (~/.gomi)
	// This represents the user's intended trash management approach.
	Strategy Strategy

	// Type determines which storage implementation to use.
	// While Strategy represents the user's intent, Type represents the actual
	// storage implementation being used. This separation allows for flexible
	// mapping between user configuration and internal implementation.
	Type StorageType

	// HomeTrashDir specifies a custom home trash directory
	HomeTrashDir string

	// EnableHomeFallback enables fallback to home trash when external trash fails
	EnableHomeFallback bool

	// ForceHomeTrash forces using home trash even for external devices
	ForceHomeTrash bool

	// SkipMountPointFind skips finding mount points for external trash dirs
	SkipMountPointFind bool

	// PreservePaths preserves original path structure in trash
	PreservePaths bool

	// UseCompression enables compression for trashed files
	UseCompression bool

	// HomeFallback enables fallback to home trash when external trash fails
	HomeFallback bool

	// History contains history-related configuration
	History config.History

	// For backwards compatibility
	TrashDir string
	RunID    string
}

// NewDefaultConfig creates a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		Strategy:           StrategyAuto,
		Type:               StorageTypeXDG,
		EnableHomeFallback: true,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.HomeTrashDir != "" {
		if !filepath.IsAbs(c.HomeTrashDir) {
			return fmt.Errorf("home trash directory must be an absolute path: %s", c.HomeTrashDir)
		}
	}

	// If no trash dir is specified, set default based on type
	if c.HomeTrashDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		if c.Strategy == StrategyLegacy {
			c.HomeTrashDir = filepath.Join(home, ".gomi")
		} else {
			c.HomeTrashDir = filepath.Join(home, ".local", "share", "Trash")
		}
	}

	return nil
}
