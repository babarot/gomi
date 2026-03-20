package xdg

import (
	"strings"
	"testing"
	"time"
)

func TestNewInfo(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, info *TrashInfo)
	}{
		{
			name: "valid trashinfo",
			input: "[Trash Info]\nPath=/home/user/file.txt\nDeletionDate=2024-01-15T10:30:00\n",
			check: func(t *testing.T, info *TrashInfo) {
				if info.Path != "/home/user/file.txt" {
					t.Errorf("Path = %q, want %q", info.Path, "/home/user/file.txt")
				}
				if info.OriginalName != "file.txt" {
					t.Errorf("OriginalName = %q, want %q", info.OriginalName, "file.txt")
				}
				want := time.Date(2024, 1, 15, 10, 30, 0, 0, time.Local)
				if !info.DeletionDate.Equal(want) {
					t.Errorf("DeletionDate = %v, want %v", info.DeletionDate, want)
				}
			},
		},
		{
			name: "encoded path with spaces",
			input: "[Trash Info]\nPath=/home/user/my%20file.txt\nDeletionDate=2024-01-15T10:30:00\n",
			check: func(t *testing.T, info *TrashInfo) {
				if info.Path != "/home/user/my file.txt" {
					t.Errorf("Path = %q, want %q", info.Path, "/home/user/my file.txt")
				}
			},
		},
		{
			name: "with comments and blank lines",
			input: "# comment\n\n[Trash Info]\n\nPath=/tmp/file\nDeletionDate=2024-01-01T00:00:00\n",
			check: func(t *testing.T, info *TrashInfo) {
				if info.Path != "/tmp/file" {
					t.Errorf("Path = %q, want %q", info.Path, "/tmp/file")
				}
			},
		},
		{
			name:    "missing header",
			input:   "Path=/tmp/file\nDeletionDate=2024-01-01T00:00:00\n",
			wantErr: true,
		},
		{
			name:    "missing path",
			input:   "[Trash Info]\nDeletionDate=2024-01-01T00:00:00\n",
			wantErr: true,
		},
		{
			name:    "missing deletion date",
			input:   "[Trash Info]\nPath=/tmp/file\n",
			wantErr: true,
		},
		{
			name:    "invalid date format",
			input:   "[Trash Info]\nPath=/tmp/file\nDeletionDate=2024/01/01 00:00:00\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := NewInfo(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, info)
			}
		})
	}
}

func TestEncodeTrashPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple path", "/home/user/file.txt", "/home/user/file.txt"},
		{"path with spaces", "/home/user/my file.txt", "/home/user/my%20file.txt"},
		{"path with special chars", "/home/user/file (1).txt", "/home/user/file%20%281%29.txt"},
		{"relative path", "Documents/file.txt", "Documents/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeTrashPath(tt.input); got != tt.want {
				t.Errorf("encodeTrashPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrashInfo_GetAbsolutePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		mountRoot string
		want      string
	}{
		{"absolute path", "/home/user/file.txt", "", "/home/user/file.txt"},
		{"relative with mount root", "Documents/file.txt", "/media/usb", "/media/usb/Documents/file.txt"},
		{"relative without mount root", "Documents/file.txt", "", "Documents/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &TrashInfo{Path: tt.path, MountRoot: tt.mountRoot}
			if got := info.GetAbsolutePath(); got != tt.want {
				t.Errorf("GetAbsolutePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrashInfo_GetRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		mountRoot string
		want      string
	}{
		{"no mount root", "/home/user/file.txt", "", "/home/user/file.txt"},
		{"already relative", "file.txt", "/media/usb", "file.txt"},
		{"absolute with mount root", "/media/usb/Documents/file.txt", "/media/usb", "Documents/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &TrashInfo{Path: tt.path, MountRoot: tt.mountRoot}
			if got := info.GetRelativePath(); got != tt.want {
				t.Errorf("GetRelativePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTrashInfo_SetMountRoot(t *testing.T) {
	info := &TrashInfo{}
	info.setMountRoot("/media/usb")
	if info.MountRoot != "/media/usb" {
		t.Errorf("MountRoot = %q, want %q", info.MountRoot, "/media/usb")
	}
}
