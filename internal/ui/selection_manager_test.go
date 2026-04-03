package ui

import (
	"testing"

	"github.com/babarot/gomi/internal/trash"
)

func newTestFile(name string) File {
	return File{File: &trash.File{Name: name, TrashPath: "/trash/" + name}}
}

func TestSelectionManager_Add(t *testing.T) {
	sm := &SelectionManager{items: []File{}}

	f1 := newTestFile("a.txt")
	sm.Add(f1)
	if len(sm.items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sm.items))
	}

	// Adding duplicate should be no-op
	sm.Add(f1)
	if len(sm.items) != 1 {
		t.Errorf("expected 1 item after duplicate add, got %d", len(sm.items))
	}
}

func TestSelectionManager_Remove(t *testing.T) {
	sm := &SelectionManager{items: []File{}}

	f1 := newTestFile("a.txt")
	f2 := newTestFile("b.txt")
	sm.Add(f1)
	sm.Add(f2)

	sm.Remove(f1)
	if len(sm.items) != 1 {
		t.Fatalf("expected 1 item after remove, got %d", len(sm.items))
	}
	if sm.items[0].Name != "b.txt" {
		t.Errorf("remaining item = %q, want %q", sm.items[0].Name, "b.txt")
	}

	// Remove non-existent should be no-op
	sm.Remove(newTestFile("nonexistent.txt"))
	if len(sm.items) != 1 {
		t.Errorf("expected 1 item after remove non-existent, got %d", len(sm.items))
	}
}

func TestSelectionManager_Contains(t *testing.T) {
	sm := &SelectionManager{items: []File{}}

	f1 := newTestFile("a.txt")
	if sm.Contains(f1) {
		t.Error("should not contain f1 before adding")
	}

	sm.Add(f1)
	if !sm.Contains(f1) {
		t.Error("should contain f1 after adding")
	}
}

func TestSelectionManager_IndexOf(t *testing.T) {
	sm := &SelectionManager{items: []File{}}

	f1 := newTestFile("a.txt")
	f2 := newTestFile("b.txt")
	sm.Add(f1)
	sm.Add(f2)

	if idx := sm.IndexOf(f1); idx != 0 {
		t.Errorf("IndexOf(f1) = %d, want 0", idx)
	}
	if idx := sm.IndexOf(f2); idx != 1 {
		t.Errorf("IndexOf(f2) = %d, want 1", idx)
	}

	missing := newTestFile("missing.txt")
	if idx := sm.IndexOf(missing); idx != -1 {
		t.Errorf("IndexOf(missing) = %d, want -1", idx)
	}
}
