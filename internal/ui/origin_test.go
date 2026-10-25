package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	dir := t.TempDir()
	taken := filepath.Join(dir, "taken.txt")
	if err := os.WriteFile(taken, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path string
		want origin
	}{
		{taken, originTaken},
		{filepath.Join(dir, "free.txt"), originBack},
		{filepath.Join(dir, "gone", "a.txt"), originGone},
	}
	for _, tt := range tests {
		if got := checkOrigin(tt.path); got != tt.want {
			t.Errorf("checkOrigin(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
