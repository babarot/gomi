package xdg

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseTrashInfoFile(t *testing.T) {
	dir := t.TempDir()
	infoFile := filepath.Join(dir, "test.trashinfo")

	content := "[Trash Info]\nPath=/home/user/test.txt\nDeletionDate=2024-06-15T10:30:00\n"
	if err := os.WriteFile(infoFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	orphan, err := ParseTrashInfoFile(infoFile)
	if err != nil {
		t.Fatalf("ParseTrashInfoFile() error = %v", err)
	}
	if orphan.OriginalPath != "/home/user/test.txt" {
		t.Errorf("OriginalPath = %q, want %q", orphan.OriginalPath, "/home/user/test.txt")
	}
	if orphan.DeletedAt.Year() != 2024 || orphan.DeletedAt.Month() != 6 || orphan.DeletedAt.Day() != 15 {
		t.Errorf("DeletedAt = %v, unexpected", orphan.DeletedAt)
	}
	if orphan.TrashInfoPath != infoFile {
		t.Errorf("TrashInfoPath = %q, want %q", orphan.TrashInfoPath, infoFile)
	}
}

func TestParseTrashInfoFile_InvalidDate(t *testing.T) {
	dir := t.TempDir()
	infoFile := filepath.Join(dir, "bad.trashinfo")
	if err := os.WriteFile(infoFile, []byte("Path=/foo\nDeletionDate=not-a-date\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ParseTrashInfoFile(infoFile)
	if err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestParseTrashInfoFile_NonExistent(t *testing.T) {
	_, err := ParseTrashInfoFile("/nonexistent/file.trashinfo")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// newOrphanTestTrash makes an empty trash with files/ and info/.
func newOrphanTestTrash(t *testing.T) (root, filesDir, infoDir string) {
	t.Helper()
	root = t.TempDir()
	filesDir = filepath.Join(root, "files")
	infoDir = filepath.Join(root, "info")
	for _, d := range []string{filesDir, infoDir} {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return root, filesDir, infoDir
}

func writeInfo(t *testing.T, infoDir, name, origPath string) {
	t.Helper()
	content := "[Trash Info]\nPath=" + origPath + "\nDeletionDate=2024-06-15T10:30:00\n"
	if err := os.WriteFile(filepath.Join(infoDir, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func orphanNames(orphans []OrphanedFile) map[string]bool {
	names := map[string]bool{}
	for _, o := range orphans {
		names[o.GetName()] = true
	}
	return names
}

// prune deletes every .trashinfo this reports, so one whose data is still in
// files/ must never be reported: without it the data cannot be restored.
func TestFindOrphanedTrashInfoFiles(t *testing.T) {
	root, filesDir, infoDir := newOrphanTestTrash(t)

	// a file and a directory that are still in the trash
	if err := os.WriteFile(filepath.Join(filesDir, "kept.txt"), []byte("kept"), 0600); err != nil {
		t.Fatal(err)
	}
	writeInfo(t, infoDir, "kept.txt.trashinfo", "/home/u/kept.txt")
	if err := os.MkdirAll(filepath.Join(filesDir, "keptdir"), 0700); err != nil {
		t.Fatal(err)
	}
	writeInfo(t, infoDir, "keptdir.trashinfo", "/home/u/keptdir")

	// metadata whose data is gone
	writeInfo(t, infoDir, "gone.txt.trashinfo", "/home/u/gone.txt")

	// not reported: a macOS resource fork, a file that is not .trashinfo,
	// a directory named like one, and an orphan that cannot be parsed
	writeInfo(t, infoDir, "._gone.txt.trashinfo", "/home/u/._gone.txt")
	if err := os.WriteFile(filepath.Join(infoDir, "notes.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(infoDir, "dir.trashinfo"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infoDir, "broken.trashinfo"), []byte("[Trash Info]\nPath=/x\nDeletionDate=bad\n"), 0600); err != nil {
		t.Fatal(err)
	}

	orphans, err := FindOrphanedTrashInfoFiles(root)
	if err != nil {
		t.Fatalf("FindOrphanedTrashInfoFiles() error = %v", err)
	}
	if len(orphans) != 1 {
		t.Fatalf("got %d orphans %v, want only gone.txt.trashinfo", len(orphans), orphanNames(orphans))
	}
	o := orphans[0]
	if o.GetName() != "gone.txt.trashinfo" {
		t.Errorf("name = %q, want gone.txt.trashinfo", o.GetName())
	}
	if want := filepath.Join(infoDir, "gone.txt.trashinfo"); o.TrashInfoPath != want {
		t.Errorf("TrashInfoPath = %q, want %q", o.TrashInfoPath, want)
	}
	if o.OriginalPath != "/home/u/gone.txt" {
		t.Errorf("OriginalPath = %q, want /home/u/gone.txt", o.OriginalPath)
	}
	if o.GetDeletedAt().IsZero() {
		t.Error("DeletedAt should be parsed")
	}

	// finding orphans only reads: nothing is removed
	for _, p := range []string{
		filepath.Join(filesDir, "kept.txt"),
		filepath.Join(infoDir, "kept.txt.trashinfo"),
		filepath.Join(infoDir, "gone.txt.trashinfo"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s should still exist: %v", p, err)
		}
	}
}

// Items gomi itself puts in the trash are never orphans.
func TestFindOrphanedTrashInfoFiles_AfterPut(t *testing.T) {
	s, dataDir := newTestStorage(t)
	for _, name := range []string{"a.txt", "b.txt"} {
		src := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(src, []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
		if err := s.Put(src); err != nil {
			t.Fatal(err)
		}
	}
	orphans, err := FindOrphanedTrashInfoFiles(filepath.Join(dataDir, "Trash"))
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 0 {
		t.Errorf("got orphans %v, want none", orphanNames(orphans))
	}
}

func TestFindOrphanedTrashInfoFiles_NoInfoDir(t *testing.T) {
	if _, err := FindOrphanedTrashInfoFiles(t.TempDir()); err == nil {
		t.Error("expected an error when info/ does not exist")
	}
}

// The spec pairs files/NAME with info/NAME.trashinfo by name: the entry in
// files/ is the trashed item, whatever a symlink there points to. A trashed
// symlink whose target is gone is still in the trash, so its .trashinfo is
// not an orphan; prune would otherwise remove it and the link could no
// longer be restored.
func TestFindOrphanedTrashInfoFiles_DanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}

	root, filesDir, infoDir := newOrphanTestTrash(t)
	if err := os.Symlink(filepath.Join(root, "nowhere"), filepath.Join(filesDir, "link")); err != nil {
		t.Fatal(err)
	}
	writeInfo(t, infoDir, "link.trashinfo", "/home/u/link")

	orphans, err := FindOrphanedTrashInfoFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 0 {
		t.Errorf("got orphans %v, want none: the link is still in files/", orphanNames(orphans))
	}
}
