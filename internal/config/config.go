// Package config provides configuration management for the application.
// It handles loading, validation, and access to application settings.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/babarot/gomi/internal/utils/shell"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v2"
)

// Config represents the root configuration structure that holds all application settings.
type Config struct {
	Core    Core          `yaml:"core"`
	UI      UI            `yaml:"ui"`
	History History       `yaml:"history"`
	Logging LoggingConfig `yaml:"logging"`
}

// Core contains core application settings that affect fundamental behaviors.
type Core struct {
	// Trash contains trash management configuration
	Trash TrashConfig `yaml:"trash"`

	// HomeFallback enables fallback to home trash when external trash fails
	HomeFallback bool `yaml:"home_fallback"`

	// Restore contains restore-specific settings
	Restore RestoreConfig `yaml:"restore"`

	// PermanentDelete contains permanent deletion feature settings
	PermanentDelete PermanentDeleteConfig `yaml:"permanent_delete"`

	// Deprecated
	TrashDir string `yaml:"trash_dir" validate:"deprecated"`
}

// TrashConfig holds trash-specific settings for managing deleted files
type TrashConfig struct {
	// Strategy determines which trash implementation to use:
	// - "auto": automatically detect and use both XDG and legacy if available
	// - "xdg": strictly follow XDG trash specification
	// - "legacy": use gomi's legacy trash format
	Strategy string `yaml:"strategy" validate:"validStrategy|allowEmpty"`

	// GomiDir specifies the trash directory for legacy mode
	GomiDir string `yaml:"gomi_dir" validate:"omitempty,validDirPath"`
}

// RestoreConfig defines settings for file restoration behavior
type RestoreConfig struct {
	// Confirm asks for confirmation before restoring
	Confirm bool `yaml:"confirm"`

	// Verbose enables detailed output during restore
	Verbose bool `yaml:"verbose"`
}

// PermanentDeleteConfig defines settings for file permanent deletion behavior
type PermanentDeleteConfig struct {
	Enable bool `yaml:"enable"`
}

type LoggingConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Level    string         `yaml:"level" validate:"oneof=debug info warn error"`
	Rotation RotationConfig `yaml:"rotation"`
}

type RotationConfig struct {
	MaxSize  string `yaml:"max_size" validate:"validSize|allowEmpty"`
	MaxFiles int    `yaml:"max_files" validate:"gte=0"`
}

// UI holds all user interface related configurations
type UI struct {
	// Density controls the compactness of the UI (compact or spacious)
	Density string `yaml:"density" validate:"omitempty,oneof=compact spacious"`

	// Style contains UI styling configuration
	Style StyleConfig `yaml:"style"`

	// ExitMessage is displayed when the application exits
	ExitMessage string `yaml:"exit_message"`

	// Preview configures file preview behavior
	Preview PreviewConfig `yaml:"preview"`

	// Paginator specifies the type of pagination (dots or arabic)
	Paginator string `yaml:"paginator_type" validate:"omitempty,oneof=dots arabic"`
}

// StyleConfig defines the visual styling of the UI
type StyleConfig struct {
	ListView       ListViewConfig   `yaml:"list_view"`
	DetailView     DetailViewConfig `yaml:"detail_view"`
	DeletionDialog string           `yaml:"deletion_dialog" validate:"validColorCode|allowEmpty"`
}

// ListViewConfig configures the file list view
type ListViewConfig struct {
	IndentOnSelect bool   `yaml:"indent_on_select"`
	Cursor         string `yaml:"cursor" validate:"validColorCode|allowEmpty"`
	Selected       string `yaml:"selected" validate:"validColorCode|allowEmpty"`
}

// DetailViewConfig configures the detailed file view
type DetailViewConfig struct {
	Border      string            `yaml:"border" validate:"validColorCode|allowEmpty"`
	InfoPane    InfoPaneConfig    `yaml:"info_pane"`
	PreviewPane PreviewPaneConfig `yaml:"preview_pane"`
}

// InfoPaneConfig configures the information panel styles
type InfoPaneConfig struct {
	DeletedFrom ColorConfig `yaml:"deleted_from"`
	DeletedAt   ColorConfig `yaml:"deleted_at"`
}

// PreviewPaneConfig configures the file preview panel
type PreviewPaneConfig struct {
	Border string      `yaml:"border" validate:"validColorCode|allowEmpty"`
	Size   ColorConfig `yaml:"size"`
	Scroll ColorConfig `yaml:"scroll"`
}

// ColorConfig defines foreground and background colors
type ColorConfig struct {
	Foreground string `yaml:"fg" validate:"validColorCode|allowEmpty"`
	Background string `yaml:"bg" validate:"validColorCode|allowEmpty"`
}

// PreviewConfig configures file preview behavior
type PreviewConfig struct {
	// SyntaxHighlight enables syntax highlighting in preview
	SyntaxHighlight bool `yaml:"syntax_highlight"`

	// Colorscheme specifies the syntax highlighting theme
	Colorscheme string `yaml:"colorscheme"`

	// DirectoryCommand is the command used to list directory contents
	DirectoryCommand string `yaml:"directory_command"`
}

// History configures history management and filtering
type History struct {
	Include IncludeConfig `yaml:"include"`
	Exclude ExcludeConfig `yaml:"exclude"`
}

// IncludeConfig specifies which files to include in history
type IncludeConfig struct {
	// Period specifies how many days of history to include
	Period int `yaml:"within_days"`
}

// ExcludeConfig specifies which files to exclude from history
type ExcludeConfig struct {
	// Files lists specific filenames to exclude
	Files []string `yaml:"files"`

	// Patterns lists regex patterns to exclude
	Patterns []string `yaml:"patterns"`

	// Globs lists glob patterns to exclude
	Globs []string `yaml:"globs"`

	// Size specifies size-based exclusions
	Size SizeConfig `yaml:"size"`
}

// SizeConfig specifies size-based filtering
type SizeConfig struct {
	// Min is the minimum file size (e.g., "10MB")
	Min string `yaml:"min" validate:"validSize|allowEmpty"`

	// Max is the maximum file size (e.g., "1GB")
	Max string `yaml:"max" validate:"validSize|allowEmpty"`
}

// Load loads the configuration from the specified path.
// If path is empty, it uses the default configuration path.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return nil, fmt.Errorf("failed to get default config path: %w", err)
		}
	}
	slog.Debug("config path found", "config-path", path)

	// Ensure config file exists
	cfg, err := ensureConfig(path)
	if err != nil {
		return nil, err
	}

	// Load and parse config
	if err := cfg.load(path); err != nil {
		return nil, err
	}

	// Validate config
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Expand paths and environment variables
	if err := cfg.expandPaths(); err != nil {
		return nil, err
	}

	// Set default value if empty
	cfg.setDefault()

	slog.Debug("config successfully loaded")
	return cfg, nil
}

// DefaultConfigPath returns the default configuration file path following XDG spec
func DefaultConfigPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	} else {
		slog.Debug("config follows XDG", "XDG_CONFIG_HOME", configDir)
	}
	return filepath.Join(configDir, "gomi", "config.yaml"), nil
}

// load reads and parses the configuration file
func (c *Config) load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	slog.Debug("config unmarshaled into struct")
	return nil
}

// ensureConfig ensures the config file exists, creating it with defaults if missing
func ensureConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); err == nil {
		return &Config{}, nil
	}

	// Create config directory if needed
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config
	cfg := NewDefaultConfig()

	// Write default config
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write default config: %w", err)
	}

	slog.Debug("ensure config")
	return cfg, nil
}

// validate performs configuration validation
func (c *Config) validate() error {
	validate := validator.New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.Split(fld.Tag.Get("yaml"), ",")[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Register custom validators
	_ = validate.RegisterValidation("validStrategy", validateStrategy)
	_ = validate.RegisterValidation("allowEmpty", validateAllowEmpty)
	_ = validate.RegisterValidation("validSize", validateSize)
	_ = validate.RegisterValidation("validColorCode", validateColorCode)
	_ = validate.RegisterValidation("deprecated", validateDeprecated)
	_ = validate.RegisterValidation("validDirPath", validateDirPath)

	if err := validate.Struct(c); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, err := range validationErrors {
				return fmt.Errorf("validation error: Field %s, %q is invalid", err.Field(), err.Value())
			}
		}
		return fmt.Errorf("unexpected validation error: %w", err)
	}

	slog.Debug("config validate done")
	return nil
}

// expandPaths expands all file paths in the configuration
func (c *Config) expandPaths() error {
	// Expand GomiDir path
	if c.Core.Trash.GomiDir != "" {
		expanded, err := shell.ExpandHome(c.Core.Trash.GomiDir)
		if err != nil {
			return fmt.Errorf("failed to expand GomiDir path: %w", err)
		}
		c.Core.Trash.GomiDir = expanded
	}

	return nil
}

func (c *Config) setDefault() {
	// Set color for deletion confirmation dialog
	// Since deletion is a potentially destructive operation, use a distinctive color to emphasize caution
	if c.UI.Style.DeletionDialog == "" {
		c.UI.Style.DeletionDialog = "205"
	}
}
