package trash

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	t.Run("valid config with storage", func(t *testing.T) {
		cfg := Config{
			HomeTrashDir: t.TempDir(),
		}
		m, err := NewManager(cfg, func(m *Manager) {
			m.storages = append(m.storages, &mockStorage{storageType: StorageTypeXDG})
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.strategy != StrategyXDG {
			t.Errorf("strategy = %v, want %v", m.strategy, StrategyXDG)
		}
	})

	t.Run("no storage backend", func(t *testing.T) {
		cfg := Config{
			HomeTrashDir: t.TempDir(),
		}
		_, err := NewManager(cfg)
		if err == nil {
			t.Fatal("expected error for no storage backend")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := Config{
			HomeTrashDir: "relative/path",
		}
		_, err := NewManager(cfg)
		if err == nil {
			t.Fatal("expected error for invalid config")
		}
	})
}

func TestManager_Put(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Run("success on first storage", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{},
			},
		}
		if err := m.Put(testFile); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fallback to second storage", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{putErr: errors.New("fail"), trashes: []string{"/trash1"}},
				&mockStorage{trashes: []string{"/trash2"}},
			},
		}
		if err := m.Put(testFile); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("all storages fail", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{putErr: errors.New("fail1"), trashes: []string{"/trash1"}},
				&mockStorage{putErr: errors.New("fail2"), trashes: []string{"/trash2"}},
			},
		}
		if err := m.Put(testFile); err == nil {
			t.Fatal("expected error when all storages fail")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{&mockStorage{}},
		}
		if err := m.Put(filepath.Join(tmpDir, "nonexistent")); err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})
}

func TestManager_List(t *testing.T) {
	t.Run("merges files from all storages", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{
					files:   []*File{{Name: "a"}},
					trashes: []string{"/trash1"},
				},
				&mockStorage{
					files:   []*File{{Name: "b"}},
					trashes: []string{"/trash2"},
				},
			},
		}
		files, err := m.List()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("got %d files, want 2", len(files))
		}
	})

	t.Run("partial failure returns files from healthy storage", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{
					files:   []*File{{Name: "a"}},
					trashes: []string{"/trash1"},
				},
				&mockStorage{
					listErr: errors.New("fail"),
					trashes: []string{"/trash2"},
				},
			},
		}
		files, err := m.List()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Errorf("got %d files, want 1", len(files))
		}
	})

	t.Run("all storages fail", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{
					listErr: errors.New("fail"),
					trashes: []string{"/trash1"},
				},
			},
		}
		_, err := m.List()
		if err == nil {
			t.Fatal("expected error when all storages fail")
		}
	})
}

func TestManager_Restore(t *testing.T) {
	t.Run("restores to original path", func(t *testing.T) {
		tmpDir := t.TempDir()
		m := &Manager{
			storages: []Storage{
				&mockStorage{trashes: []string{"/trash"}},
			},
		}
		file := &File{
			Name:         "test.txt",
			TrashPath:    "/trash/test.txt",
			OriginalPath: filepath.Join(tmpDir, "test.txt"),
		}
		err := m.Restore(file, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("file not in any storage", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{trashes: []string{"/trash"}},
			},
		}
		file := &File{
			Name:         "test.txt",
			TrashPath:    "/other/test.txt",
			OriginalPath: "/tmp/test.txt",
		}
		err := m.Restore(file, "")
		if err == nil {
			t.Fatal("expected error for unknown storage")
		}
	})

	t.Run("destination already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		existing := filepath.Join(tmpDir, "exists.txt")
		if err := os.WriteFile(existing, []byte("data"), 0644); err != nil {
			t.Fatal(err)
		}

		m := &Manager{
			storages: []Storage{
				&mockStorage{trashes: []string{"/trash"}},
			},
		}
		file := &File{
			Name:         "exists.txt",
			TrashPath:    "/trash/exists.txt",
			OriginalPath: existing,
		}
		err := m.Restore(file, "")
		if !IsFileExists(err) {
			t.Errorf("expected ErrFileExists, got %v", err)
		}
	})
}

func TestManager_Remove(t *testing.T) {
	t.Run("removes from correct storage", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{trashes: []string{"/trash"}},
			},
		}
		file := &File{TrashPath: "/trash/test.txt"}
		if err := m.Remove(file); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("file not in any storage", func(t *testing.T) {
		m := &Manager{
			storages: []Storage{
				&mockStorage{trashes: []string{"/trash"}},
			},
		}
		file := &File{TrashPath: "/other/test.txt"}
		if err := m.Remove(file); err == nil {
			t.Fatal("expected error for unknown storage")
		}
	})
}
