package trash

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/config"
)

// Config holds the unified configuration for trash management
type Config struct {
	// Type determines which storage implementation to use
	Type StorageType

	// HomeTrashDir specifies a custom home trash directory
	HomeTrashDir string

	// EnableHomeFallback enables fallback to home trash when external trash fails
	EnableHomeFallback bool

	// UseXDG enables XDG-compliant trash specification
	UseXDG bool

	// ForceHomeTrash forces using home trash even for external devices
	ForceHomeTrash bool

	// SkipMountPointFind skips finding mount points for external trash dirs
	SkipMountPointFind bool

	// PreservePaths preserves original path structure in trash
	PreservePaths bool

	// UseCompression enables compression for trashed files
	UseCompression bool

	// Verbose enables detailed output
	Verbose bool

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
		Type:               StorageTypeXDG,
		EnableHomeFallback: true,
		UseXDG:             true,
		Verbose:            false,
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

		if c.UseXDG {
			c.HomeTrashDir = filepath.Join(home, ".local", "share", "Trash")
		} else {
			c.HomeTrashDir = filepath.Join(home, ".gomi")
		}
	}

	return nil
}

// WithXDG enables XDG-compliant trash storage
func (c *Config) WithXDG() *Config {
	c.Type = StorageTypeXDG
	c.UseXDG = true
	return c
}

// WithLegacy enables legacy trash storage
func (c *Config) WithLegacy(h config.History) *Config {
	c.Type = StorageTypeLegacy
	c.UseXDG = false
	c.History = h
	return c
}

// FromConfig creates a trash Config from a gomi config
func FromConfig(cfg config.Config) *Config {
	return &Config{
		Type:               StorageTypeXDG,
		HomeTrashDir:       cfg.Core.TrashDir,
		EnableHomeFallback: cfg.Core.HomeFallback,
		UseXDG:             cfg.Core.UseXDG,
		Verbose:            cfg.Core.Verbose,
		History:            cfg.History,
		TrashDir:           cfg.Core.TrashDir,
	}
}
