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

func TestConfig_Validate_Theme(t *testing.T) {
	for _, name := range append(ThemeNames, "") {
		cfg := NewDefaultConfig()
		cfg.UI.Theme = name
		if err := cfg.validate(); err != nil {
			t.Errorf("theme %q: %v", name, err)
		}
	}
	cfg := NewDefaultConfig()
	cfg.UI.Theme = "solarized"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for an unknown theme")
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
	if err := os.WriteFile(configPath, []byte("core:\n  trash:\n    strategy: auto\n"), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

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

func TestParseHeight(t *testing.T) {
	tests := []struct {
		in      string
		n       int
		percent bool
		ok      bool
	}{
		{"30", 30, false, true},
		{"40%", 40, true, true},
		{"100%", 100, true, true},
		{"101%", 0, false, false},
		{"0", 0, false, false},
		{"-3", 0, false, false},
		{"auto", 0, false, false},
		{"%", 0, false, false},
	}
	for _, tt := range tests {
		n, percent, ok := ParseHeight(tt.in)
		if n != tt.n || percent != tt.percent || ok != tt.ok {
			t.Errorf("ParseHeight(%q) = %d, %v, %v; want %d, %v, %v", tt.in, n, percent, ok, tt.n, tt.percent, tt.ok)
		}
	}
}

func TestConfig_Validate_Height(t *testing.T) {
	for _, h := range []string{"auto", "30", "40%", ""} {
		cfg := NewDefaultConfig()
		cfg.UI.Height = h
		if err := cfg.validate(); err != nil {
			t.Errorf("height %q: %v", h, err)
		}
	}
	cfg := NewDefaultConfig()
	cfg.UI.Height = "full"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for height full")
	}
}

func TestConfig_Validate_PreviewPosition(t *testing.T) {
	for _, pos := range []string{"auto", "right", "below", "none", ""} {
		cfg := NewDefaultConfig()
		cfg.UI.Preview.Position = pos
		if err := cfg.validate(); err != nil {
			t.Errorf("position %q: %v", pos, err)
		}
	}
	cfg := NewDefaultConfig()
	cfg.UI.Preview.Position = "left"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for position left")
	}
}

func TestConfig_Validate_Scope(t *testing.T) {
	for _, scope := range []string{"", "auto", "here", "all"} {
		cfg := NewDefaultConfig()
		cfg.UI.Scope = scope
		if err := cfg.validate(); err != nil {
			t.Errorf("scope %q: %v", scope, err)
		}
	}
	cfg := NewDefaultConfig()
	cfg.UI.Scope = "there"
	if err := cfg.validate(); err == nil {
		t.Error("expected validation error for scope there")
	}
}

func TestConfig_Validate_Views(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.UI.View = "places"
	cfg.UI.Views = []string{"places", "recent"}
	if err := cfg.validate(); err != nil {
		t.Errorf("valid views: %v", err)
	}
	for _, bad := range []func(c *Config){
		func(c *Config) { c.UI.View = "tree" },
		func(c *Config) { c.UI.Views = []string{"recent", "tree"} },
	} {
		cfg := NewDefaultConfig()
		bad(cfg)
		if err := cfg.validate(); err == nil {
			t.Error("expected validation error for an unknown view")
		}
	}
}

// setHome points the home directory at a temporary one for the test
func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

// writeConfig writes content as a config file and returns its path
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_CreatesDefaultConfig(t *testing.T) {
	home := setHome(t)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	path := filepath.Join(home, "xdg", "gomi", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default config not written to %s: %v", path, err)
	}
	if want := filepath.Join(home, ".gomi"); cfg.Core.Trash.GomiDir != want {
		t.Errorf("GomiDir = %q, want %q", cfg.Core.Trash.GomiDir, want)
	}
	if cfg.Core.Trash.Strategy != "auto" {
		t.Errorf("Strategy = %q, want %q", cfg.Core.Trash.Strategy, "auto")
	}
	if !cfg.Core.Restore.Confirm {
		t.Error("Restore.Confirm should be true by default")
	}

	// Loading again reads the file just written
	again, err := Load(path)
	if err != nil {
		t.Fatalf("Load(%s) error = %v", path, err)
	}
	if again.Core.Trash.GomiDir != cfg.Core.Trash.GomiDir || !again.Core.Restore.Confirm {
		t.Errorf("reloaded config differs: %+v", again.Core)
	}
}

func TestLoad_ExistingFileIsKept(t *testing.T) {
	setHome(t)
	content := "core:\n  trash:\n    strategy: legacy\n"
	path := writeConfig(t, content)

	if _, err := Load(path); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Errorf("config file was rewritten: %q", data)
	}
}

func TestLoad_Values(t *testing.T) {
	home := setHome(t)
	path := writeConfig(t, `core:
  trash:
    strategy: legacy
    home_fallback: false
    gomi_dir: ~/my-trash
    forbidden_paths:
      - /important
  restore:
    confirm: false
    verbose: true
  permanent_delete:
    enable: true
ui:
  theme: nord
  view: places
history:
  include:
    within_days: 7
  exclude:
    files:
      - secret.txt
    size:
      max: 1GB
logging:
  level: warn
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	tr := cfg.Core.Trash
	if tr.Strategy != "legacy" {
		t.Errorf("Strategy = %q, want legacy", tr.Strategy)
	}
	if tr.HomeFallback {
		t.Error("HomeFallback should be false")
	}
	if want := home + "/my-trash"; tr.GomiDir != want {
		t.Errorf("GomiDir = %q, want %q", tr.GomiDir, want)
	}
	if len(tr.ForbiddenPaths) != 1 || tr.ForbiddenPaths[0] != "/important" {
		t.Errorf("ForbiddenPaths = %v, want [/important]", tr.ForbiddenPaths)
	}
	if cfg.Core.Restore.Confirm || !cfg.Core.Restore.Verbose {
		t.Errorf("Restore = %+v, want confirm false, verbose true", cfg.Core.Restore)
	}
	if !cfg.Core.PermanentDelete.Enable {
		t.Error("PermanentDelete.Enable should be true")
	}
	if cfg.UI.Theme != "nord" || cfg.UI.View != "places" {
		t.Errorf("UI theme, view = %q, %q, want nord, places", cfg.UI.Theme, cfg.UI.View)
	}
	if cfg.History.Include.Period != 7 {
		t.Errorf("Period = %d, want 7", cfg.History.Include.Period)
	}
	if ex := cfg.History.Exclude; len(ex.Files) != 1 || ex.Files[0] != "secret.txt" || ex.Size.Max != "1GB" {
		t.Errorf("Exclude = %+v, want files [secret.txt], max 1GB", ex)
	}
	if cfg.Logging.Level != "warn" {
		t.Errorf("Logging.Level = %q, want warn", cfg.Logging.Level)
	}
}

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string // in the error
	}{
		{"malformed yaml", "core:\n  trash: [\n", "parse"},
		{"invalid strategy", "core:\n  trash:\n    strategy: cloud\n", "strategy"},
		{"invalid size", "history:\n  exclude:\n    size:\n      max: huge\n", "max"},
		// trash_dir is retired: it errors instead of silently choosing
		// another trash directory than the user wrote
		{"retired trash_dir", "core:\n  trash_dir: /tmp/old-trash\n", "trash_dir"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHome(t)
			cfg, err := Load(writeConfig(t, tt.content))
			if err == nil {
				t.Fatalf("Load() = %+v, want an error", cfg)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load() error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestExpandPaths(t *testing.T) {
	home := setHome(t)
	t.Setenv("GOMI_TEST_DIR", "trash-from-env")

	tests := []struct {
		name    string
		gomiDir string
		want    string
	}{
		{"empty stays empty", "", ""},
		{"tilde", "~/.gomi", home + "/.gomi"},
		{"env var", "~/$GOMI_TEST_DIR", home + "/trash-from-env"},
		{"absolute", "/var/gomi", "/var/gomi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Core: Core{Trash: TrashConfig{GomiDir: tt.gomiDir}}}
			if err := c.expandPaths(); err != nil {
				t.Fatalf("expandPaths() error = %v", err)
			}
			if c.Core.Trash.GomiDir != tt.want {
				t.Errorf("GomiDir = %q, want %q", c.Core.Trash.GomiDir, tt.want)
			}
		})
	}
}
