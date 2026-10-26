package xdg

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/utils/fs"
)

const testRunID = "test-run"

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
		RunID:          testRunID,
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
	if files[0].RunID != testRunID {
		t.Errorf("RunID = %q, want %q", files[0].RunID, testRunID)
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

// gomi puts its arguments in parallel (cli.Put), so files with the same name
// from different directories (rm .astro node_modules/.astro) race for the
// same name in the trash. Every one of them must get a name of its own.
func TestStorage_Put_ConcurrentCollision(t *testing.T) {
	s, _ := newTestStorage(t)

	const n = 20
	srcs := make([]string, n)
	for i := range srcs {
		srcs[i] = filepath.Join(t.TempDir(), "dup.txt")
		if err := os.WriteFile(srcs[i], []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i, src := range srcs {
		wg.Go(func() {
			<-start
			errs[i] = s.Put(src)
		})
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("Put() #%d error = %v", i, err)
		}
	}
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != n {
		t.Errorf("List() returned %d files, want %d", len(files), n)
	}
}

// When the trash cannot tell whether a name is taken (files/ is not
// searchable), Put must fail instead of trying new names forever.
func TestStorage_Put_UncheckableName(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root is never denied permission")
	}
	s, dataDir := newTestStorage(t)

	srcFile := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(srcFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	filesDir := filepath.Join(dataDir, "Trash", "files")
	if err := os.Chmod(filesDir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filesDir, 0755) })

	done := make(chan error, 1)
	go func() { done <- s.Put(srcFile) }()
	select {
	case err := <-done:
		if !errors.Is(err, os.ErrPermission) {
			t.Errorf("Put() error = %v, want a permission error", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Put() did not return")
	}

	if _, err := os.Stat(srcFile); err != nil {
		t.Errorf("source file should stay in place: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, "Trash", "info"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("info dir has %d entries, want none", len(entries))
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
		RunID:        testRunID,
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
	if loaded.RunID != info.RunID {
		t.Errorf("RunID = %q, want %q", loaded.RunID, info.RunID)
	}
}

func TestTrashInfo_Save_WithoutRunID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}

	infoPath := filepath.Join(t.TempDir(), "test.trashinfo")
	info := &TrashInfo{Path: "/home/user/a.txt", DeletionDate: fixedTime()}
	if err := info.Save(infoPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Without a run ID the file is just what the spec defines
	b, err := os.ReadFile(infoPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "[Trash Info]\nPath=/home/user/a.txt\nDeletionDate=" + fixedTime().Format(timeFormat) + "\n"
	if string(b) != want {
		t.Errorf("content = %q, want %q", b, want)
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

func TestStorage_List_SizeKnown(t *testing.T) {
	s, _ := newTestStorage(t)

	srcDir := t.TempDir()
	file := filepath.Join(srcDir, "a.txt")
	if err := os.WriteFile(file, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(srcDir, "d")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{file, dir} {
		if err := s.Put(p); err != nil {
			t.Fatal(err)
		}
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		// stat gives a file's size, not a directory's total
		if want := !f.IsDir; f.SizeKnown != want {
			t.Errorf("%s: SizeKnown = %v, want %v", f.Name, f.SizeKnown, want)
		}
	}
}

// newTestXDGStorage is newTestStorage with the concrete type, so that tests
// can turn HomeFallback off and look at the trash locations.
func newTestXDGStorage(t *testing.T, homeFallback bool) (*Storage, string) {
	t.Helper()
	s, dataDir := newTestStorage(t)
	xs, ok := s.(*Storage)
	if !ok {
		t.Fatalf("NewStorage() returned %T, want *Storage", s)
	}
	xs.config.HomeFallback = homeFallback
	return xs, dataDir
}

// putOne trashes a new file with the given content and returns its source
// path and the one item the trash then lists.
func putOne(t *testing.T, s trash.Storage, name, content string) (string, *trash.File) {
	t.Helper()
	src := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(src); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("List() returned %d files, want 1", len(files))
	}
	return src, files[0]
}

func assertContent(t *testing.T, path, want string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(b) != want {
		t.Errorf("%s: content = %q, want %q", path, b, want)
	}
}

func assertNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("%s should not exist (err = %v)", path, err)
	}
}

// With no destination given, the file goes back to where it was trashed from.
func TestStorage_Restore_EmptyDstUsesOriginalPath(t *testing.T) {
	s, _ := newTestStorage(t)
	src, f := putOne(t, s, "back.txt", "back home")

	if err := s.Restore(f, ""); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	assertContent(t, src, "back home")
	assertNotExist(t, f.TrashPath)
	assertNotExist(t, infoPathForFile(f.TrashPath))
}

// The data file was removed from files/ behind gomi's back: Restore must
// fail and must not create anything at the destination.
func TestStorage_Restore_MissingTrashFile(t *testing.T) {
	s, _ := newTestStorage(t)
	src, f := putOne(t, s, "vanished.txt", "x")

	if err := os.Remove(f.TrashPath); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(f, src); err == nil {
		t.Fatal("Restore() should fail when the trashed file is missing")
	}
	assertNotExist(t, src)
	// the metadata is left for prune to judge, not removed by a failed restore
	if _, err := os.Stat(infoPathForFile(f.TrashPath)); err != nil {
		t.Errorf(".trashinfo should be kept: %v", err)
	}
}

// When the destination cannot be created, the trashed file and its
// .trashinfo stay where they are and the item is still listed.
func TestStorage_Restore_FailureKeepsTrashItem(t *testing.T) {
	s, _ := newTestStorage(t)
	_, f := putOne(t, s, "keep.txt", "precious")

	// the destination's parent is a regular file, so it cannot be a directory
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("blocker"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(f, filepath.Join(blocker, "keep.txt")); err == nil {
		t.Fatal("Restore() should fail when the destination directory cannot be made")
	}

	assertContent(t, blocker, "blocker")
	assertContent(t, f.TrashPath, "precious")
	if _, err := os.Stat(infoPathForFile(f.TrashPath)); err != nil {
		t.Errorf(".trashinfo should be kept: %v", err)
	}
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].TrashPath != f.TrashPath {
		t.Errorf("List() after failed restore = %v, want the item still in the trash", files)
	}
}

// Storage.Restore does not ask: the CLI asks the user before restoring onto
// an existing name (cli.restoreFile) and only then calls it. Restoring onto
// an existing file replaces that file with the trashed one.
func TestStorage_Restore_ReplacesExistingFile(t *testing.T) {
	s, _ := newTestStorage(t)
	src, f := putOne(t, s, "same.txt", "from trash")

	if err := os.WriteFile(src, []byte("newer"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Restore(f, src); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	assertContent(t, src, "from trash")
	assertNotExist(t, f.TrashPath)
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("List() returned %d after restore, want 0", len(files))
	}
}

// A directory cannot be renamed onto a non-empty directory. Without the
// copy fallback, Restore must fail and leave both directories untouched.
func TestStorage_Restore_OntoNonEmptyDirWithoutFallback(t *testing.T) {
	s, _ := newTestXDGStorage(t, false)

	srcDir := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "old.txt"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(srcDir); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	files, err := s.List()
	if err != nil || len(files) != 1 {
		t.Fatalf("List() = %v, %v; want one item", files, err)
	}
	f := files[0]

	// a new directory of the same name was made in the meantime
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "new.txt"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Restore(f, srcDir); err == nil {
		t.Fatal("Restore() onto a non-empty directory should fail without fallback")
	}
	assertContent(t, filepath.Join(srcDir, "new.txt"), "new")
	assertNotExist(t, filepath.Join(srcDir, "old.txt"))
	assertContent(t, filepath.Join(f.TrashPath, "old.txt"), "old")
	if _, err := os.Stat(infoPathForFile(f.TrashPath)); err != nil {
		t.Errorf(".trashinfo should be kept: %v", err)
	}
}

func TestSelectTrashLocation(t *testing.T) {
	s, _ := newTestXDGStorage(t, false)
	home := s.homeTrash
	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	// t.TempDir and XDG_DATA_HOME are on the same device in the test
	same, err := isOnSameDevice(src, home.root)
	if err != nil || !same {
		t.Skipf("temp dir and test trash are not on one device (same=%v, err=%v)", same, err)
	}

	t.Run("same device goes to home trash", func(t *testing.T) {
		loc, err := s.selectTrashLocation(src)
		if err != nil {
			t.Fatalf("selectTrashLocation() error = %v", err)
		}
		if loc != home {
			t.Errorf("got %s, want home trash %s", loc.root, home.root)
		}
	})

	// a home trash that cannot be compared stands for one on another device
	unreachable := &trashLocation{root: filepath.Join(t.TempDir(), "missing"), isHome: true}
	ext := &trashLocation{root: t.TempDir(), mountRoot: "/"}

	t.Run("external trash on the file's device", func(t *testing.T) {
		s := &Storage{homeTrash: unreachable, externalTrashes: []*trashLocation{ext}}
		loc, err := s.selectTrashLocation(src)
		if err != nil {
			t.Fatalf("selectTrashLocation() error = %v", err)
		}
		if loc != ext {
			t.Errorf("got %s, want external trash %s", loc.root, ext.root)
		}
	})

	t.Run("no matching trash with home fallback", func(t *testing.T) {
		s := &Storage{homeTrash: unreachable, config: trash.Config{HomeFallback: true}}
		loc, err := s.selectTrashLocation(src)
		if err != nil {
			t.Fatalf("selectTrashLocation() error = %v", err)
		}
		if loc != unreachable {
			t.Errorf("got %s, want home trash", loc.root)
		}
	})

	t.Run("no matching trash without home fallback", func(t *testing.T) {
		s := &Storage{homeTrash: unreachable}
		if _, err := s.selectTrashLocation(src); !errors.Is(err, trash.ErrCrossDevice) {
			t.Errorf("selectTrashLocation() error = %v, want ErrCrossDevice", err)
		}
	})
}

// When no trash can take the file and home fallback is off, Put fails and
// the file stays where it is, with no .trashinfo left behind.
func TestStorage_Put_CrossDeviceWithoutFallback(t *testing.T) {
	s, dataDir := newTestXDGStorage(t, false)
	infoDir := s.homeTrash.infoDir
	// make the home trash impossible to compare against, as if on another device
	s.homeTrash = &trashLocation{
		root:     filepath.Join(dataDir, "missing"),
		filesDir: s.homeTrash.filesDir,
		infoDir:  infoDir,
		isHome:   true,
	}

	src := filepath.Join(t.TempDir(), "stay.txt")
	if err := os.WriteFile(src, []byte("stay"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(src); !errors.Is(err, trash.ErrCrossDevice) {
		t.Fatalf("Put() error = %v, want ErrCrossDevice", err)
	}
	assertContent(t, src, "stay")
	entries, err := os.ReadDir(infoDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("info dir has %d entries, want none", len(entries))
	}
}

// Without XDG_DATA_HOME the home trash is ~/.local/share/Trash.
func TestInitHomeTrash_DefaultsToHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG trash is not used on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	loc, err := (&Storage{}).initHomeTrash()
	if err != nil {
		t.Fatalf("initHomeTrash() error = %v", err)
	}
	want := filepath.Join(home, ".local", "share", "Trash")
	if loc.root != want {
		t.Errorf("root = %q, want %q", loc.root, want)
	}
	if !loc.isHome || loc.mountRoot != "" {
		t.Errorf("isHome = %v, mountRoot = %q; want true, empty", loc.isHome, loc.mountRoot)
	}
	for _, dir := range []string{loc.filesDir, loc.infoDir} {
		fi, err := os.Stat(dir)
		if err != nil {
			t.Errorf("%s not created: %v", dir, err)
			continue
		}
		if fi.Mode().Perm()&0077 != 0 {
			t.Errorf("%s mode = %o, want no access for group and others", dir, fi.Mode().Perm())
		}
	}
}

// A trash that cannot be created makes NewStorage fail rather than return a
// storage that would lose files on Put.
func TestNewStorage_HomeTrashUncreatable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG trash is not used on Windows")
	}
	for _, tc := range []struct {
		name    string
		blocked string // path made a regular file before NewStorage
	}{
		{"data dir is a file", ""},
		{"files dir is a file", filepath.Join("Trash", "files")},
		{"info dir is a file", filepath.Join("Trash", "info")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := filepath.Join(t.TempDir(), "data")
			blocked := filepath.Join(dataDir, tc.blocked)
			if err := os.MkdirAll(filepath.Dir(blocked), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(blocked, []byte("not a dir"), 0644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_DATA_HOME", dataDir)

			if _, err := NewStorage(trash.Config{ForceHomeTrash: true}); err == nil {
				t.Error("NewStorage() should fail when the home trash cannot be created")
			}
			assertContent(t, blocked, "not a dir")
		})
	}
}

// Opening the storage again keeps what is already in the trash.
func TestNewStorage_KeepsExistingItems(t *testing.T) {
	s, dataDir := newTestStorage(t)
	src, _ := putOne(t, s, "kept.txt", "still here")

	s2, err := NewStorage(trash.Config{ForceHomeTrash: true, HomeFallback: true})
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	files, err := s2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].OriginalPath != src {
		t.Fatalf("List() = %v, want the item trashed before", files)
	}
	if want := filepath.Join(dataDir, "Trash", "files", "kept.txt"); files[0].TrashPath != want {
		t.Errorf("TrashPath = %q, want %q", files[0].TrashPath, want)
	}
	assertContent(t, files[0].TrashPath, "still here")
}

// Scanning the real mount points for external trashes only looks; the home
// trash stays first and the storage stays usable.
func TestNewStorage_ScansExternalTrashes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG trash is not used on Windows")
	}
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	s, err := NewStorage(trash.Config{HomeFallback: true})
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	xs, ok := s.(*Storage)
	if !ok {
		t.Fatalf("NewStorage() returned %T, want *Storage", s)
	}
	trashes := s.Info().Trashes
	if want := filepath.Join(dataDir, "Trash"); len(trashes) == 0 || trashes[0] != want {
		t.Fatalf("Trashes = %v, want %s first", trashes, want)
	}
	for _, ext := range xs.externalTrashes {
		if ext.isHome || ext.mountRoot == "" {
			t.Errorf("external trash %s: isHome = %v, mountRoot = %q", ext.root, ext.isHome, ext.mountRoot)
		}
		if !isValidExternalTrash(ext.root) {
			t.Errorf("external trash %s does not pass isValidExternalTrash", ext.root)
		}
	}
}

func TestFindAllTrashDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG trash is not used on Windows")
	}
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)

	dirs, err := FindAllTrashDirectories()
	if err != nil {
		t.Fatalf("FindAllTrashDirectories() error = %v", err)
	}
	if want := filepath.Join(dataDir, "Trash"); len(dirs) == 0 || dirs[0] != want {
		t.Fatalf("dirs = %v, want %s first", dirs, want)
	}
	seen := map[string]bool{}
	for _, d := range dirs {
		if seen[d] {
			t.Errorf("%s listed twice", d)
		}
		seen[d] = true
	}
	// prune reads info/ of each; the home one must be there to read
	if _, err := FindOrphanedTrashInfoFiles(dirs[0]); err != nil {
		t.Errorf("FindOrphanedTrashInfoFiles(home) error = %v", err)
	}
}

// A dangling symlink is trashed as the link itself (rename moves the link),
// so the trash must list it, or it cannot be restored.
func TestStorage_PutAndList_DanglingSymlink(t *testing.T) {
	s, _ := newTestStorage(t)
	link := filepath.Join(t.TempDir(), "dangling")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone"), link); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(link); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].OriginalPath != link {
		t.Fatalf("List() = %v, want the dangling link", files)
	}
}

// When the copy fallback cannot remove all of the source, the trash holds
// the only complete copy, so Put keeps its .trashinfo and reports what is
// left behind.
func TestStorage_Put_SourceRemainsIsListed(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root is never denied permission")
	}
	s, _ := newTestXDGStorage(t, true)

	// rename fails since the parent is read-only; RemoveAll then removes
	// the files inside src and fails on src itself
	parent := filepath.Join(t.TempDir(), "locked")
	src := filepath.Join(parent, "src")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(parent, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0755) })

	if err := s.Put(src); !errors.Is(err, fs.ErrSourceRemains) {
		t.Fatalf("Put() error = %v, want fs.ErrSourceRemains", err)
	}

	files, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("List() returned %d files, want 1", len(files))
	}
	assertContent(t, filepath.Join(files[0].TrashPath, "a.txt"), "a")
	assertContent(t, filepath.Join(files[0].TrashPath, "sub", "b.txt"), "b")
}
