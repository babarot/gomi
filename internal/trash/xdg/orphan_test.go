package xdg

import (
	"os"
	"path/filepath"
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
