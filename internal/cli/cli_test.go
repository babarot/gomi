package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/utils/log"
)

func TestExpandPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.txt", "foo.txt"},
		{"./foo.txt", "foo.txt"},
		{"foo/../bar.txt", "bar.txt"},
		{"foo/./bar.txt", "foo/bar.txt"},
		{"/absolute/path", "/absolute/path"},
		{"", "."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := expandPath(tt.input)
			if err != nil {
				t.Fatalf("expandPath(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandWindowsPaths(t *testing.T) {
	// Without actual glob matches, args should be returned as-is
	args := []string{"noexist*.xyz", "plain.txt"}
	got := expandWindowsPaths(args)
	if len(got) != 2 {
		t.Fatalf("expandWindowsPaths() returned %d items, want 2", len(got))
	}
	if got[1] != "plain.txt" {
		t.Errorf("expandWindowsPaths()[1] = %q, want %q", got[1], "plain.txt")
	}
}

func TestExpandWindowsPaths_WithMatches(t *testing.T) {
	dir := t.TempDir()

	// Create temp files to match glob
	for _, name := range []string{"a.tmp", "b.tmp"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	args := []string{dir + "/*.tmp"}
	got := expandWindowsPaths(args)
	if len(got) != 2 {
		t.Fatalf("expandWindowsPaths() returned %d items, want 2", len(got))
	}
}

func TestDetermineLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  log.Level
	}{
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"unknown", log.DebugLevel},
		{"", log.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := determineLogLevel(tt.input)
			if got != tt.want {
				t.Errorf("determineLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsForbiddenPath(t *testing.T) {
	cli := &CLI{
		config: &config.Config{
			Core: config.Core{
				Trash: config.TrashConfig{
					ForbiddenPaths: []string{
						"/",
						"/etc",
						"/usr",
						"$HOME/.gomi",
					},
				},
			},
		},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/etc", true},
		{"/etc/hosts", true},     // sub-path of /etc
		{"/usr", true},           // exact match
		{"/usr/local/bin", true}, // sub-path of /usr
		{"/tmp/foo", false},
		{"/home/user/file.txt", false},
		{"relative/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := cli.isForbiddenPath(tt.path)
			if got != tt.want {
				t.Errorf("isForbiddenPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSyncStringSlice(t *testing.T) {
	s := &syncStringSlice{}

	// Empty initially
	if got := s.Get(); len(got) != 0 {
		t.Errorf("empty slice: Get() returned %d items", len(got))
	}

	// Append and verify
	s.Append("a")
	s.Append("b")
	s.Append("c")

	got := s.Get()
	if len(got) != 3 {
		t.Fatalf("Get() returned %d items, want 3", len(got))
	}
	if got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("Get() = %v, want [a b c]", got)
	}

	// Verify Get() returns a copy (modifying returned slice doesn't affect original)
	got[0] = "modified"
	original := s.Get()
	if original[0] != "a" {
		t.Error("Get() should return a copy, not a reference")
	}
}

func TestSyncStringSlice_Concurrent(t *testing.T) {
	s := &syncStringSlice{}
	done := make(chan struct{})

	// Concurrent writes
	for i := 0; i < 100; i++ {
		go func() {
			s.Append("item")
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}

	if got := s.Get(); len(got) != 100 {
		t.Errorf("concurrent Append: got %d items, want 100", len(got))
	}
}
