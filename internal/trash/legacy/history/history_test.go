package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
)

func TestFile_Getters(t *testing.T) {
	now := time.Now()
	f := File{
		Name:      "test.txt",
		To:        "/trash/test.txt",
		Timestamp: now,
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

func TestNew(t *testing.T) {
	t.Run("with home dir", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix-specific test")
		}
		h := New("/custom/home", config.History{})
		if h.home != "/custom/home" {
			t.Errorf("home = %q, want %q", h.home, "/custom/home")
		}
		if h.path != "/custom/home/history.json" {
			t.Errorf("path = %q, want %q", h.path, "/custom/home/history.json")
		}
	})

	t.Run("empty home uses default", func(t *testing.T) {
		home, _ := os.UserHomeDir()
		t.Setenv("HOME", home)
		h := New("", config.History{})
		want := filepath.Join(home, ".gomi")
		if h.home != want {
			t.Errorf("home = %q, want %q", h.home, want)
		}
	})
}

func TestHistory_FindByID(t *testing.T) {
	h := History{
		Files: []File{
			{ID: "abc", Name: "file1.txt"},
			{ID: "def", Name: "file2.txt"},
		},
	}

	t.Run("found", func(t *testing.T) {
		f := h.FindByID("abc")
		if f == nil {
			t.Fatal("expected non-nil")
		}
		if f.Name != "file1.txt" {
			t.Errorf("Name = %q, want %q", f.Name, "file1.txt")
		}
	})

	t.Run("not found", func(t *testing.T) {
		if f := h.FindByID("xyz"); f != nil {
			t.Errorf("expected nil, got %v", f)
		}
	})
}

func TestHistory_RemoveByPath(t *testing.T) {
	h := History{
		Files: []File{
			{Name: "a", To: "/trash/a"},
			{Name: "b", To: "/trash/b"},
			{Name: "c", To: "/trash/c"},
		},
	}

	h.RemoveByPath("/trash/b")
	if len(h.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(h.Files))
	}
	for _, f := range h.Files {
		if f.To == "/trash/b" {
			t.Error("file b should have been removed")
		}
	}
}

func TestHistory_Add(t *testing.T) {
	h := History{}
	h.Add(File{Name: "new.txt"})
	if len(h.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(h.Files))
	}
	if h.Files[0].Name != "new.txt" {
		t.Errorf("Name = %q, want %q", h.Files[0].Name, "new.txt")
	}
}

func TestHistory_SetVersion(t *testing.T) {
	h := History{}
	h.setVersion()
	if h.Version != historyVersion {
		t.Errorf("Version = %d, want %d", h.Version, historyVersion)
	}

	// Should not overwrite existing version
	h.Version = 99
	h.setVersion()
	if h.Version != 99 {
		t.Errorf("Version = %d, want 99", h.Version)
	}
}

func TestHistory_FileInfo(t *testing.T) {
	h := New("/tmp/gomi", config.History{})
	f, err := h.FileInfo("run1", "/home/user/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "test.txt" {
		t.Errorf("Name = %q, want %q", f.Name, "test.txt")
	}
	if f.RunID != "run1" {
		t.Errorf("RunID = %q, want %q", f.RunID, "run1")
	}
	if f.ID == "" {
		t.Error("ID should not be empty")
	}
	if f.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestFile_String(t *testing.T) {
	f := File{Name: "test.txt", ID: "abc"}
	s := f.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestHistory_Open(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("empty file", func(t *testing.T) {
		h := New(tmpDir, config.History{})
		if err := h.Open(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(h.Files) != 0 {
			t.Errorf("expected 0 files, got %d", len(h.Files))
		}
	})

	t.Run("valid history file", func(t *testing.T) {
		dir := t.TempDir()
		historyData := History{
			Version: 1,
			Files: []File{
				{Name: "test.txt", ID: "abc", From: "/home/test.txt", To: filepath.Join(dir, "test.txt")},
			},
		}
		data, _ := json.Marshal(historyData)
		os.WriteFile(filepath.Join(dir, Filename), data, 0644)

		h := New(dir, config.History{})
		if err := h.Open(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(h.Files) != 1 {
			t.Errorf("expected 1 file, got %d", len(h.Files))
		}
	})
}

func TestHistory_Update(t *testing.T) {
	dir := t.TempDir()
	h := New(dir, config.History{})
	h.Open()

	newFiles := []File{
		{Name: "a.txt", ID: "id1"},
		{Name: "b.txt", ID: "id2"},
	}
	if err := h.Update(newFiles); err != nil {
		t.Fatal(err)
	}
	if len(h.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(h.Files))
	}
	if h.Version != historyVersion {
		t.Errorf("Version = %d, want %d", h.Version, historyVersion)
	}

	// Verify file was written
	data, err := os.ReadFile(filepath.Join(dir, Filename))
	if err != nil {
		t.Fatal(err)
	}
	var saved History
	json.Unmarshal(data, &saved)
	if len(saved.Files) != 2 {
		t.Errorf("saved %d files, want 2", len(saved.Files))
	}
}

func TestHistory_Save(t *testing.T) {
	dir := t.TempDir()
	h := New(dir, config.History{})
	h.Files = []File{{Name: "test.txt", ID: "abc"}}
	h.Version = 1

	if err := h.Save(); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, Filename))
	var saved History
	json.Unmarshal(data, &saved)
	if len(saved.Files) != 1 {
		t.Errorf("saved %d files, want 1", len(saved.Files))
	}
}

func TestHistory_Remove(t *testing.T) {
	dir := t.TempDir()
	h := New(dir, config.History{})
	h.Files = []File{
		{Name: "a.txt", ID: "id1"},
		{Name: "b.txt", ID: "id2"},
		{Name: "c.txt", ID: "id3"},
	}
	h.Version = 1

	if err := h.Remove(File{ID: "id2"}); err != nil {
		t.Fatal(err)
	}
	if len(h.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(h.Files))
	}
	for _, f := range h.Files {
		if f.ID == "id2" {
			t.Error("id2 should have been removed")
		}
	}
}

func TestHistory_Backup(t *testing.T) {
	dir := t.TempDir()
	h := New(dir, config.History{})
	h.Files = []File{{Name: "test.txt"}}
	h.Version = 1

	if err := h.backup(); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(dir, Filename+".backup")
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file not created: %v", err)
	}
}
