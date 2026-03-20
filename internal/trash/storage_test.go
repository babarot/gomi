package trash

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestStorageType_String(t *testing.T) {
	tests := []struct {
		st   StorageType
		want string
	}{
		{StorageTypeXDG, "xdg"},
		{StorageTypeLegacy, "legacy"},
		{StorageType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.st.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFile_Getters(t *testing.T) {
	now := time.Now()
	f := &File{
		Name:      "test.txt",
		TrashPath: "/trash/test.txt",
		DeletedAt: now,
	}

	if got := f.GetName(); got != "test.txt" {
		t.Errorf("GetName() = %q, want %q", got, "test.txt")
	}
	if got := f.GetPath(); got != "/trash/test.txt" {
		t.Errorf("GetPath() = %q, want %q", got, "/trash/test.txt")
	}
	if got := f.GetDeletedAt(); got != now {
		t.Errorf("GetDeletedAt() = %v, want %v", got, now)
	}
}

func TestFile_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	existing := filepath.Join(tmpDir, "exists.txt")
	os.WriteFile(existing, []byte("hello"), 0644)

	tests := []struct {
		name      string
		trashPath string
		want      bool
	}{
		{"exists", existing, true},
		{"not exists", filepath.Join(tmpDir, "nope.txt"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{TrashPath: tt.trashPath}
			if got := f.Exists(); got != tt.want {
				t.Errorf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFile_RequiresAdmin(t *testing.T) {
	tmpDir := t.TempDir()

	writable := filepath.Join(tmpDir, "writable.txt")
	os.WriteFile(writable, []byte("hello"), 0644)

	readonly := filepath.Join(tmpDir, "readonly.txt")
	os.WriteFile(readonly, []byte("hello"), 0444)

	tests := []struct {
		name      string
		trashPath string
		want      bool
	}{
		{"writable", writable, false},
		{"readonly", readonly, true},
		{"nonexistent", filepath.Join(tmpDir, "nope"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{TrashPath: tt.trashPath}
			if got := f.RequiresAdmin(); got != tt.want {
				t.Errorf("RequiresAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFile_GetOriginalPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tests := []struct {
		name         string
		originalPath string
		mountRoot    string
		want         string
	}{
		{"absolute path", "/home/user/file.txt", "", "/home/user/file.txt"},
		{"relative with mount root", "Documents/file.txt", "/media/usb", "/media/usb/Documents/file.txt"},
		{"relative without mount root", "Documents/file.txt", "", "Documents/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{OriginalPath: tt.originalPath, MountRoot: tt.mountRoot}
			if got := f.GetOriginalPath(); got != tt.want {
				t.Errorf("GetOriginalPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFile_GetRelativePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	tests := []struct {
		name         string
		originalPath string
		mountRoot    string
		want         string
	}{
		{"no mount root", "/home/user/file.txt", "", "/home/user/file.txt"},
		{"relative path input", "file.txt", "/media/usb", "file.txt"},
		{"absolute with mount root", "/media/usb/Documents/file.txt", "/media/usb", "Documents/file.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{OriginalPath: tt.originalPath, MountRoot: tt.mountRoot}
			if got := f.GetRelativePath(); got != tt.want {
				t.Errorf("GetRelativePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFile_Storage(t *testing.T) {
	f := &File{}
	if got := f.GetStorage(); got != nil {
		t.Errorf("expected nil storage, got %v", got)
	}

	// We can't easily create a mock Storage here since it's in the same package,
	// but we can verify SetStorage/GetStorage round-trips with nil
	f.SetStorage(nil)
	if got := f.GetStorage(); got != nil {
		t.Errorf("expected nil after SetStorage(nil), got %v", got)
	}
}
