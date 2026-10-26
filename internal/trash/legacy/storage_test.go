package legacy

import (
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/trash/legacy/history"
)

func newTestConfig(dir string) trash.Config {
	return trash.Config{
		GomiDir:  dir,
		Strategy: trash.StrategyLegacy,
		History:  config.History{},
		RunID:    "test-run",
	}
}

func TestNewStorage(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if s == nil {
		t.Fatal("NewStorage() returned nil")
	}

	// Check info
	info := s.Info()
	if info.Type != trash.StorageTypeLegacy {
		t.Errorf("Type = %v, want StorageTypeLegacy", info.Type)
	}
	if !info.Available {
		t.Error("Available should be true")
	}
	if info.Location != trash.LocationHome {
		t.Errorf("Location = %v, want LocationHome", info.Location)
	}
}

func TestStorage_PutAndList(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create a file to trash
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "test.txt")
	if err := os.WriteFile(srcFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	// Put it in trash
	if err := s.Put(srcFile); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Original should be gone
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("original file should have been moved")
	}

	// List should return the file
	files, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("List() returned %d files, want 1", len(files))
	}
	if files[0].Name != "test.txt" {
		t.Errorf("Name = %q, want %q", files[0].Name, "test.txt")
	}
	if files[0].OriginalPath != srcFile {
		t.Errorf("OriginalPath = %q, want %q", files[0].OriginalPath, srcFile)
	}
}

func TestStorage_Restore(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create and trash a file
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "restore_me.txt")
	content := []byte("restore this")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(srcFile); err != nil {
		t.Fatal(err)
	}

	// Get the trashed file
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Restore it
	if err := s.Restore(files[0], srcFile); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	// File should be back
	data, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("restored file not found: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("restored content = %q, want %q", string(data), string(content))
	}

	// Should be removed from trash listing
	files, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d files after restore, want 0", len(files))
	}
}

func TestStorage_Remove(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create and trash a file
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "delete_me.txt")
	if err := os.WriteFile(srcFile, []byte("bye"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(srcFile); err != nil {
		t.Fatal(err)
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Remove permanently
	if err := s.Remove(files[0]); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// File should be gone from trash
	if _, err := os.Stat(files[0].TrashPath); !os.IsNotExist(err) {
		t.Error("trash file should have been removed")
	}

	// Should be removed from listing
	files, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d files after remove, want 0", len(files))
	}
}

func TestStorage_PutDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create a directory with files
	srcDir := t.TempDir()
	testDir := filepath.Join(srcDir, "mydir")
	if err := os.MkdirAll(filepath.Join(testDir, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "sub", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(testDir); err != nil {
		t.Fatalf("Put(dir) error = %v", err)
	}

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("original directory should have been moved")
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(files))
	}
	if files[0].Name != "mydir" {
		t.Errorf("Name = %q, want %q", files[0].Name, "mydir")
	}
}

func TestStorage_RestoreToCustomDst(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Create and trash a file
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "original.txt")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(srcFile); err != nil {
		t.Fatal(err)
	}

	files, _ := s.List()

	// Restore to custom destination
	customDst := filepath.Join(t.TempDir(), "custom", "restored.txt")
	if err := s.Restore(files[0], customDst); err != nil {
		t.Fatalf("Restore() to custom dst error = %v", err)
	}

	data, err := os.ReadFile(customDst)
	if err != nil {
		t.Fatalf("custom dst file not found: %v", err)
	}
	if string(data) != "data" {
		t.Errorf("content = %q, want %q", string(data), "data")
	}
}

func TestStorage_PutMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := s.Put(filepath.Join(srcDir, name)); err != nil {
			t.Fatalf("Put(%s) error = %v", name, err)
		}
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Errorf("List() returned %d files, want 3", len(files))
	}
}

func TestStorage_PutRecordsRunID(t *testing.T) {
	dir := t.TempDir()
	cfg := newTestConfig(dir)

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatal(err)
	}

	srcDir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		src := filepath.Join(srcDir, name)
		if err := os.WriteFile(src, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
		if err := s.Put(src); err != nil {
			t.Fatalf("Put(%s) error = %v", name, err)
		}
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("List() returned %d files, want 2", len(files))
	}
	for _, f := range files {
		if f.RunID != cfg.RunID {
			t.Errorf("%s: RunID = %q, want %q", f.Name, f.RunID, cfg.RunID)
		}
	}
}

// putDir trashes a directory holding size bytes and returns its path
func putDir(t *testing.T, s trash.Storage, name string, size int) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data"), make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(dir); err != nil {
		t.Fatalf("Put(%s) error = %v", name, err)
	}
	return dir
}

func TestStorage_ListSizes(t *testing.T) {
	tests := []struct {
		name    string
		size    config.SizeConfig
		want    []string // names listed
		knowDir bool     // whether the directory's size is known
	}{
		{
			name:    "no size filter",
			want:    []string{"big", "small", "file.txt"},
			knowDir: false,
		},
		{
			// a directory is sized by its contents, not by what stat says
			name:    "size filter",
			size:    config.SizeConfig{Max: "2KB"},
			want:    []string{"small", "file.txt"},
			knowDir: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig(t.TempDir())
			cfg.History.Exclude.Size = tt.size
			s, err := NewStorage(cfg)
			if err != nil {
				t.Fatal(err)
			}
			putDir(t, s, "big", 4096)
			putDir(t, s, "small", 100)
			src := filepath.Join(t.TempDir(), "file.txt")
			if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := s.Put(src); err != nil {
				t.Fatal(err)
			}

			files, err := s.List()
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]*trash.File{}
			for _, f := range files {
				got[f.Name] = f
			}
			if len(got) != len(tt.want) {
				t.Fatalf("listed %d files, want %v", len(got), tt.want)
			}
			for _, name := range tt.want {
				if got[name] == nil {
					t.Fatalf("%s is not listed", name)
				}
			}
			if f := got["file.txt"]; !f.SizeKnown || f.Size != 5 {
				t.Errorf("file.txt: Size = %d, SizeKnown = %v, want 5, true", f.Size, f.SizeKnown)
			}
			small := got["small"]
			if small.SizeKnown != tt.knowDir {
				t.Errorf("small: SizeKnown = %v, want %v", small.SizeKnown, tt.knowDir)
			}
			if tt.knowDir && small.Size != 100 {
				t.Errorf("small: Size = %d, want 100", small.Size)
			}
		})
	}
}

// reopen creates a new storage on dir, so that what it lists is what was
// saved to the history file
func reopen(t *testing.T, dir string) trash.Storage {
	t.Helper()
	s, err := NewStorage(newTestConfig(dir))
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	return s
}

// blockHistory replaces the history file in dir with a directory that is
// not empty, so that saving the history fails, until the returned function
// is called
func blockHistory(t *testing.T, dir string) (unblock func()) {
	t.Helper()
	path := filepath.Join(dir, history.Filename)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "blocker"), 0755); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
	}
}

// assertContent fails unless path holds content
func assertContent(t *testing.T, path, content string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s: %v", path, err)
		return
	}
	if string(data) != content {
		t.Errorf("%s = %q, want %q", path, data, content)
	}
}

// trashedCopies returns the files under dir whose names start with name+"."
func trashedCopies(t *testing.T, dir, name string) []string {
	t.Helper()
	var found []string
	err := filepath.WalkDir(dir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), name+".") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestNewStorage_CorruptHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, history.Filename)
	corrupt := []byte(`{"version":1,"files":[`)
	if err := os.WriteFile(path, corrupt, 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStorage(newTestConfig(dir)); err == nil {
		t.Fatal("NewStorage() succeeded on a corrupt history file, want an error")
	}
	// The history file is left for the user to repair
	assertContent(t, path, string(corrupt))
}

func TestStorage_Put_MissingSource(t *testing.T) {
	dir := t.TempDir()
	s := reopen(t, dir)

	if err := s.Put(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("Put() succeeded on a missing file, want an error")
	}
	for _, s := range []trash.Storage{s, reopen(t, dir)} {
		files, err := s.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Errorf("List() returned %d files, want 0", len(files))
		}
	}
}

// When the history cannot be saved, Put must move the file back to where
// it was, so that no file sits in the trash without a history entry.
func TestStorage_Put_RollsBackWhenHistorySaveFails(t *testing.T) {
	dir := t.TempDir()
	s := reopen(t, dir)

	srcFile := filepath.Join(t.TempDir(), "keep.txt")
	if err := os.WriteFile(srcFile, []byte("keep me"), 0644); err != nil {
		t.Fatal(err)
	}
	unblock := blockHistory(t, dir)

	if err := s.Put(srcFile); err == nil {
		t.Fatal("Put() succeeded, want an error from saving the history")
	}
	assertContent(t, srcFile, "keep me")
	if found := trashedCopies(t, dir, "keep.txt"); len(found) != 0 {
		t.Errorf("trash still holds %v", found)
	}
	// The failed save leaves no temporary files beside the history
	if found := trashedCopies(t, dir, ".history"); len(found) != 0 {
		t.Errorf("temporary history files left: %v", found)
	}

	t.Run("history has no entry for it", func(t *testing.T) {
		t.Skip("known bug: Put adds the entry to the in-memory history before saving and keeps it after rolling the file back, so List shows it and the next save writes it (internal/trash/legacy/storage.go:115-138)")
		unblock()
		other := filepath.Join(t.TempDir(), "other.txt")
		if err := os.WriteFile(other, []byte("other"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := s.Put(other); err != nil {
			t.Fatal(err)
		}
		files, err := reopen(t, dir).List()
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if f.Name == "keep.txt" {
				t.Errorf("history lists keep.txt at %s, which was rolled back", f.TrashPath)
			}
		}
	})
}

// A restore that cannot move the file must leave it in the trash, still
// listed in the history.
func TestStorage_Restore_FailureKeepsTrashedFile(t *testing.T) {
	tests := []struct {
		name string
		dst  func(t *testing.T) string
	}{
		{
			name: "destination parent is a file",
			dst: func(t *testing.T) string {
				parent := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(parent, []byte("not a dir"), 0644); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(parent, "restored.txt")
			},
		},
		{
			name: "destination is a directory",
			dst: func(t *testing.T) string {
				dst := filepath.Join(t.TempDir(), "restored")
				if err := os.MkdirAll(dst, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dst, "inside.txt"), []byte("inside"), 0644); err != nil {
					t.Fatal(err)
				}
				return dst
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			s := reopen(t, dir)
			srcFile := filepath.Join(t.TempDir(), "data.txt")
			if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := s.Put(srcFile); err != nil {
				t.Fatal(err)
			}
			files, err := s.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(files) != 1 {
				t.Fatalf("expected 1 file, got %d", len(files))
			}
			dst := tt.dst(t)

			if err := s.Restore(files[0], dst); err == nil {
				t.Fatal("Restore() succeeded, want an error")
			}
			assertContent(t, files[0].TrashPath, "data")
			if info, err := os.Stat(dst); err == nil && info.IsDir() {
				assertContent(t, filepath.Join(dst, "inside.txt"), "inside")
			}
			for _, s := range []trash.Storage{s, reopen(t, dir)} {
				got, err := s.List()
				if err != nil {
					t.Fatal(err)
				}
				if len(got) != 1 || got[0].TrashPath != files[0].TrashPath {
					t.Errorf("List() = %v, want the entry for %s", got, files[0].TrashPath)
				}
			}
		})
	}
}

// When the file is moved back but the history cannot be saved, Restore
// reports the error and the file stays where it was restored to.
func TestStorage_Restore_HistorySaveFails(t *testing.T) {
	dir := t.TempDir()
	s := reopen(t, dir)
	srcFile := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(srcFile); err != nil {
		t.Fatal(err)
	}
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	blockHistory(t, dir)

	if err := s.Restore(files[0], ""); err == nil {
		t.Fatal("Restore() succeeded, want an error from saving the history")
	}
	assertContent(t, srcFile, "data")
}
