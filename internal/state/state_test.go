package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStore_KeepsWhatWasSaved(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gomi")
	s := Open(dir)
	if s.PaneWidth() != 0 || s.PaneHeight() != 0 {
		t.Fatalf("a store never written has %d, %d", s.PaneWidth(), s.PaneHeight())
	}
	if err := s.SavePaneWidth(52); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePaneHeight(14); err != nil {
		t.Fatal(err)
	}
	s = Open(dir) // the next launch
	if s.PaneWidth() != 52 || s.PaneHeight() != 14 {
		t.Errorf("width %d, height %d; want 52, 14", s.PaneWidth(), s.PaneHeight())
	}
	if err := s.SavePaneWidth(0); err != nil {
		t.Fatal(err)
	}
	if s.PaneWidth() != 0 || s.PaneHeight() != 14 {
		t.Errorf("after forgetting the width: %d, %d; want 0, 14", s.PaneWidth(), s.PaneHeight())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("the directory holds %d files, want only state.json", len(entries))
	}
}

func TestStore_BrokenFileIsNeverWritten(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := Open(dir)
	if s.PaneWidth() != 0 {
		t.Errorf("a broken file gives %d", s.PaneWidth())
	}
	if err := s.SavePaneWidth(40); err != nil {
		t.Fatal(err)
	}
	if s.PaneWidth() != 40 {
		t.Errorf("width %d after saving over a broken file, want 40", s.PaneWidth())
	}
}

func TestDefaultDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/x/state")
	if got, _ := DefaultDir(); got != filepath.Join("/x/state", "gomi") {
		t.Errorf("DefaultDir %s", got)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/u")
	t.Setenv("USERPROFILE", "/home/u") // os.UserHomeDir on Windows
	if got, _ := DefaultDir(); got != filepath.Join("/home/u", ".local", "state", "gomi") {
		t.Errorf("DefaultDir %s", got)
	}
}
