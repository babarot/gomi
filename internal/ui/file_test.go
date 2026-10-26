package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/babarot/gomi/internal/trash"
)

func TestBrowseDirectory(t *testing.T) {
	full := t.TempDir()
	if err := os.WriteFile(filepath.Join(full, "a.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	tests := []struct {
		name    string
		dir     string
		command string
		// builtin is whether the builtin listing is shown instead of want
		builtin bool
		want    string
	}{
		{name: "command lists", dir: full, command: "echo listed", want: "listed\n"},
		// cat reads stdin, as eza does when it is given no path
		{name: "command lists nothing", dir: full, command: "cat", builtin: true},
		{name: "empty directory", dir: empty, command: "true", want: ""},
		{name: "no command", dir: full, command: "", builtin: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := File{File: &trash.File{TrashPath: tt.dir}, dirListCommand: tt.command}
			got, err := f.Browse(80, 24)
			if err != nil {
				t.Fatalf("Browse() error = %v", err)
			}
			if tt.builtin {
				if !strings.HasSuffix(got, "  a.txt") {
					t.Errorf("Browse() = %q, want the builtin listing of a.txt", got)
				}
				return
			}
			if got != tt.want {
				t.Errorf("Browse() = %q, want %q", got, tt.want)
			}
		})
	}
}
