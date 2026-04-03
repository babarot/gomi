package legacy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
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
	os.MkdirAll(filepath.Join(testDir, "sub"), 0755)
	os.WriteFile(filepath.Join(testDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(testDir, "sub", "b.txt"), []byte("b"), 0644)

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
		os.WriteFile(filepath.Join(srcDir, name), []byte(name), 0644)
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
