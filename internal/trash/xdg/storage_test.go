package xdg

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
)

func newTestStorage(t *testing.T) (trash.Storage, string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("XDG trash is not used on Windows")
	}

	// Use a temp dir as XDG_DATA_HOME so we don't touch real trash
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	cfg := trash.Config{
		Strategy:       trash.StrategyXDG,
		HomeFallback:   true,
		ForceHomeTrash: true, // skip external trash scan
		History:        config.History{},
	}

	s, err := NewStorage(cfg)
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	return s, dataDir
}

func TestNewStorage_CreatesDirectories(t *testing.T) {
	s, dataDir := newTestStorage(t)
	_ = s

	trashRoot := filepath.Join(dataDir, "Trash")
	for _, sub := range []string{"files", "info"} {
		dir := filepath.Join(trashRoot, sub)
		fi, err := os.Stat(dir)
		if err != nil {
			t.Errorf("directory %q not created: %v", sub, err)
			continue
		}
		if !fi.IsDir() {
			t.Errorf("%q is not a directory", sub)
		}
	}
}

func TestStorage_Info(t *testing.T) {
	s, dataDir := newTestStorage(t)

	info := s.Info()
	if info.Type != trash.StorageTypeXDG {
		t.Errorf("Type = %v, want StorageTypeXDG", info.Type)
	}
	if !info.Available {
		t.Error("Available should be true")
	}
	wantRoot := filepath.Join(dataDir, "Trash")
	if len(info.Trashes) == 0 || info.Trashes[0] != wantRoot {
		t.Errorf("Trashes = %v, want [%s]", info.Trashes, wantRoot)
	}
}

func TestStorage_PutAndList(t *testing.T) {
	s, _ := newTestStorage(t)

	// Create a file to trash
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "hello.txt")
	if err := os.WriteFile(srcFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(srcFile); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Original should be gone
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("original file should have been removed")
	}

	files, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("List() returned %d files, want 1", len(files))
	}
	if files[0].Name != "hello.txt" {
		t.Errorf("Name = %q, want %q", files[0].Name, "hello.txt")
	}
	if files[0].OriginalPath != srcFile {
		t.Errorf("OriginalPath = %q, want %q", files[0].OriginalPath, srcFile)
	}
}

func TestStorage_Put_CreatesTrashInfo(t *testing.T) {
	s, dataDir := newTestStorage(t)

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "check_info.txt")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(srcFile); err != nil {
		t.Fatal(err)
	}

	// Verify .trashinfo file exists
	infoDir := filepath.Join(dataDir, "Trash", "info")
	entries, err := os.ReadDir(infoDir)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".trashinfo" {
			found = true
			break
		}
	}
	if !found {
		t.Error("no .trashinfo file created")
	}
}

func TestStorage_Put_CollisionHandling(t *testing.T) {
	s, _ := newTestStorage(t)

	// Trash two files with the same name from different directories
	for i := range 2 {
		srcDir := t.TempDir()
		srcFile := filepath.Join(srcDir, "dup.txt")
		if err := os.WriteFile(srcFile, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := s.Put(srcFile); err != nil {
			t.Fatalf("Put() #%d error = %v", i, err)
		}
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("List() returned %d files, want 2 (collision should create unique names)", len(files))
	}
}

func TestStorage_Restore(t *testing.T) {
	s, _ := newTestStorage(t)

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "restore_me.txt")
	content := []byte("restore this")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
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

	// Restore to original location
	if err := s.Restore(files[0], srcFile); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	data, err := os.ReadFile(srcFile)
	if err != nil {
		t.Fatalf("restored file not found: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}

	// Trash should be empty now
	files, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d after restore, want 0", len(files))
	}
}

func TestStorage_Restore_CustomDst(t *testing.T) {
	s, _ := newTestStorage(t)

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "original.txt")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Put(srcFile); err != nil {
		t.Fatal(err)
	}

	files, _ := s.List()
	customDst := filepath.Join(t.TempDir(), "subdir", "restored.txt")

	if err := s.Restore(files[0], customDst); err != nil {
		t.Fatalf("Restore() to custom dst error = %v", err)
	}

	data, err := os.ReadFile(customDst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "data" {
		t.Errorf("content = %q, want %q", string(data), "data")
	}
}

func TestStorage_Remove(t *testing.T) {
	s, _ := newTestStorage(t)

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

	trashPath := files[0].TrashPath

	if err := s.Remove(files[0]); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// File should be gone from trash
	if _, err := os.Stat(trashPath); !os.IsNotExist(err) {
		t.Error("trash file should have been removed")
	}

	// Should be empty
	files, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d after remove, want 0", len(files))
	}
}

func TestStorage_PutDirectory(t *testing.T) {
	s, _ := newTestStorage(t)

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
		t.Error("original directory should have been removed")
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
	if !files[0].IsDir {
		t.Error("IsDir should be true")
	}
}

func TestStorage_List_Empty(t *testing.T) {
	s, _ := newTestStorage(t)

	files, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d files, want 0", len(files))
	}
}

func TestStorage_List_SkipsInvalidInfo(t *testing.T) {
	s, dataDir := newTestStorage(t)

	trashRoot := filepath.Join(dataDir, "Trash")

	// Create a file in files/ without a matching .trashinfo
	if err := os.WriteFile(filepath.Join(trashRoot, "files", "orphan.txt"), []byte("orphan"), 0644); err != nil {
		t.Fatal(err)
	}

	files, err := s.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	// Should skip orphan (no valid .trashinfo)
	if len(files) != 0 {
		t.Errorf("List() returned %d files, want 0 (orphan should be skipped)", len(files))
	}
}

func TestTrashInfo_Save_And_Load(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	dir := t.TempDir()
	infoPath := filepath.Join(dir, "test.trashinfo")

	info := &TrashInfo{
		Path:         "/home/user/test file.txt",
		DeletionDate: fixedTime(),
	}

	if err := info.Save(infoPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := loadTrashInfo(infoPath)
	if err != nil {
		t.Fatalf("loadTrashInfo() error = %v", err)
	}

	if loaded.Path != info.Path {
		t.Errorf("Path = %q, want %q", loaded.Path, info.Path)
	}
	if !loaded.DeletionDate.Equal(info.DeletionDate) {
		t.Errorf("DeletionDate = %v, want %v", loaded.DeletionDate, info.DeletionDate)
	}
}

func TestTrashInfo_Save_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	infoPath := filepath.Join(dir, "existing.trashinfo")

	// Create an existing file
	if err := os.WriteFile(infoPath, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	info := &TrashInfo{
		Path:         "/tmp/file",
		DeletionDate: fixedTime(),
	}

	// Save should fail because file already exists (O_EXCL)
	err := info.Save(infoPath)
	if err == nil {
		t.Error("Save() should fail when file already exists")
	}
}

func fixedTime() time.Time {
	return time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local)
}
