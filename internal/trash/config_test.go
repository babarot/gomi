package trash

import (
	"strings"
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.Strategy != StrategyAuto {
		t.Errorf("Strategy = %v, want %v", cfg.Strategy, StrategyAuto)
	}
	if cfg.Type != StorageTypeXDG {
		t.Errorf("Type = %v, want %v", cfg.Type, StorageTypeXDG)
	}
	if !cfg.HomeFallback {
		t.Error("HomeFallback should be true by default")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name:    "valid default config",
			cfg:     NewDefaultConfig(),
			wantErr: "",
		},
		{
			name: "relative home trash dir",
			cfg: &Config{
				HomeTrashDir: "relative/path",
			},
			wantErr: "must be an absolute path",
		},
		{
			name: "absolute home trash dir",
			cfg: &Config{
				HomeTrashDir: "/tmp/trash",
			},
			wantErr: "",
		},
		{
			name: "empty home trash dir sets default for xdg",
			cfg: &Config{
				Strategy: StrategyXDG,
			},
			wantErr: "",
		},
		{
			name: "empty home trash dir sets default for legacy",
			cfg: &Config{
				Strategy: StrategyLegacy,
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestConfig_Validate_SetsDefaultDir(t *testing.T) {
	t.Run("legacy strategy sets .gomi dir", func(t *testing.T) {
		cfg := &Config{Strategy: StrategyLegacy}
		if err := cfg.Validate(); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(cfg.HomeTrashDir, ".gomi") {
			t.Errorf("HomeTrashDir = %q, want suffix .gomi", cfg.HomeTrashDir)
		}
	})

	t.Run("xdg strategy sets Trash dir", func(t *testing.T) {
		cfg := &Config{Strategy: StrategyXDG}
		if err := cfg.Validate(); err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(cfg.HomeTrashDir, "Trash") {
			t.Errorf("HomeTrashDir = %q, want suffix Trash", cfg.HomeTrashDir)
		}
	})
}
