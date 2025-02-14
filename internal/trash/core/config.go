// internal/trash/core/config.go
package core

import (
	"fmt"
	"path/filepath"

	"github.com/babarot/gomi/internal/config"
)

// Config holds the unified configuration for trash management
type Config struct {
	// Storage type and fallback settings
	Type               StorageType
	EnableHomeFallback bool

	// Common settings
	HomeTrashDir string
	Verbose      bool

	// XDG-specific settings
	UseXDG             bool
	ForceHomeTrash     bool // Force using home trash even for external devices
	SkipMountPointFind bool // Skip finding mount points for external trash dirs

	// Legacy-specific settings
	PreservePaths  bool // Preserve original path structure in trash
	UseCompression bool // Use compression for trashed files

	// for backward TODO:
	History  config.History
	TrashDir string
}

// StorageType represents the type of trash storage
type StorageType int

const (
	// StorageTypeXDG represents XDG-compliant trash storage
	StorageTypeXDG StorageType = iota

	// StorageTypeLegacy represents legacy (.gomi) trash storage
	StorageTypeLegacy
)

func (t StorageType) String() string {
	switch t {
	case StorageTypeXDG:
		return "xdg"
	case StorageTypeLegacy:
		return "legacy"
	}
	return "unknown"
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
