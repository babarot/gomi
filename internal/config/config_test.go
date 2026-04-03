package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg.Core.Trash.Strategy != "auto" {
		t.Errorf("Strategy = %q, want %q", cfg.Core.Trash.Strategy, "auto")
	}
	if !cfg.Core.Trash.HomeFallback {
		t.Error("HomeFallback should be true")
	}
	if cfg.Core.Trash.GomiDir == "" {
		t.Error("GomiDir should not be empty")
	}
	if cfg.History.Include.Period != 365 {
		t.Errorf("Period = %d, want 365", cfg.History.Include.Period)
	}
	if !cfg.Logging.Enabled {
		t.Error("Logging should be enabled by default")
	}
	if cfg.UI.Density != "spacious" {
		t.Errorf("Density = %q, want %q", cfg.UI.Density, "spacious")
	}

	// Forbidden paths must include critical system dirs
	forbiddenSet := make(map[string]bool)
	for _, p := range cfg.Core.Trash.ForbiddenPaths {
		forbiddenSet[p] = true
	}
	for _, p := range []string{"/", "/etc", "/usr", "/var", "/bin", "/sbin"} {
		if !forbiddenSet[p] {
			t.Errorf("ForbiddenPaths missing critical path %q", p)
		}
	}

	// PermanentDelete should be disabled by default
	if cfg.Core.PermanentDelete.Enable {
		t.Error("PermanentDelete.Enable should be false by default")
	}

	// Restore defaults
	if !cfg.Core.Restore.Confirm {
		t.Error("Restore.Confirm should be true by default")
	}

	// Logging rotation defaults
	if cfg.Logging.Rotation.MaxSize != "10MB" {
		t.Errorf("Rotation.MaxSize = %q, want %q", cfg.Logging.Rotation.MaxSize, "10MB")
	}
	if cfg.Logging.Rotation.MaxFiles != 3 {
		t.Errorf("Rotation.MaxFiles = %d, want 3", cfg.Logging.Rotation.MaxFiles)
	}

	// Preview defaults
	if !cfg.UI.Preview.SyntaxHighlight {
		t.Error("Preview.SyntaxHighlight should be true by default")
	}
	if cfg.UI.Paginator != "dots" {
		t.Errorf("UI.Paginator = %q, want %q", cfg.UI.Paginator, "dots")
	}

	// History exclude defaults
	if cfg.History.Exclude.Size.Max != "10GB" {
		t.Errorf("History.Exclude.Size.Max = %q, want %q", cfg.History.Exclude.Size.Max, "10GB")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix-specific test")
		}
		t.Setenv("XDG_CONFIG_HOME", "/custom/config")
		path, err := DefaultConfigPath()
		if err != nil {
			t.Fatal(err)
		}
		want := "/custom/config/gomi/config.yaml"
		if path != want {
			t.Errorf("got %q, want %q", path, want)
		}
	})

	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		path, err := DefaultConfigPath()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(path, filepath.Join(".config", "gomi", "config.yaml")) {
			t.Errorf("unexpected path: %q", path)
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	cfg := NewDefaultConfig()
	if err := cfg.validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestConfig_Validate_InvalidStrategy(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Core.Trash.Strategy = "invalid"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for invalid strategy")
	}
}

func TestConfig_Validate_InvalidSize(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.History.Exclude.Size.Min = "notasize"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for invalid size")
	}
}

func TestConfig_Validate_InvalidColor(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.UI.Style.ListView.Cursor = "notacolor"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for invalid color")
	}
}

func TestConfig_SetDefault(t *testing.T) {
	cfg := &Config{}
	cfg.setDefault()

	if cfg.UI.Style.DeletionDialog == "" {
		t.Error("DeletionDialog should have default value")
	}
	if cfg.UI.Style.ListView.FilterMatch == "" {
		t.Error("FilterMatch should have default value")
	}
	if cfg.UI.Style.ListView.FilterPrompt == "" {
		t.Error("FilterPrompt should have default value")
	}
}

func TestEnsureConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gomi", "config.yaml")

	cfg, err := ensureConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// File should be created
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestEnsureConfig_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(configPath, []byte("core:\n  trash:\n    strategy: auto\n"), 0644)

	cfg, err := ensureConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	// Returns empty config when file exists (to be loaded later)
	if cfg.Core.Trash.Strategy != "" {
		t.Errorf("expected empty config, got strategy=%q", cfg.Core.Trash.Strategy)
	}
}

func TestConfig_Load(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `core:
  trash:
    strategy: xdg
    home_fallback: true
history:
  include:
    within_days: 30
`
	os.WriteFile(configPath, []byte(content), 0644)

	cfg := &Config{}
	if err := cfg.load(configPath); err != nil {
		t.Fatal(err)
	}
	if cfg.Core.Trash.Strategy != "xdg" {
		t.Errorf("Strategy = %q, want %q", cfg.Core.Trash.Strategy, "xdg")
	}
	if cfg.History.Include.Period != 30 {
		t.Errorf("Period = %d, want 30", cfg.History.Include.Period)
	}
}
