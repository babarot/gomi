package fs

import (
	"os"
	"path/filepath"
	"testing"
)

// createTempDir creates a temporary directory for testing
func createTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gomi-fs-test-")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// createTestFile creates a test file with given content
func createTestFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func TestCreate(t *testing.T) {
	dir := createTempDir(t)
	testPath := filepath.Join(dir, "testfile.txt")

	// First create should succeed
	f, err := Create(testPath, 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	// Second create should fail (file already exists)
	_, err = Create(testPath, 0644)
	if err == nil {
		t.Fatal("Expected error when creating existing file, got nil")
	}
}

func TestMove(t *testing.T) {
	dir := createTempDir(t)
	srcPath := filepath.Join(dir, "source.txt")
	dstPath := filepath.Join(dir, "destination.txt")
	content := "test content"

	// Create source file
	createTestFile(t, srcPath, content)

	// Test successful move
	err := Move(srcPath, dstPath, false)
	if err != nil {
		t.Fatalf("Failed to move file: %v", err)
	}

	// Verify source file is gone
	_, err = os.Stat(srcPath)
	if !os.IsNotExist(err) {
		t.Fatal("Source file should not exist after move")
	}

	// Verify destination file exists with correct content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != content {
		t.Fatalf("Destination file content mismatch. Expected %q, got %q", content, dstContent)
	}

	// Test move across devices (fallback copy)
	srcPath = filepath.Join(dir, "source2.txt")
	dstPath = filepath.Join(dir, "destination2.txt")
	createTestFile(t, srcPath, content)

	err = Move(srcPath, dstPath, true)
	if err != nil {
		t.Fatalf("Failed to move file with fallback copy: %v", err)
	}

	// Verify source file is gone
	_, err = os.Stat(srcPath)
	if !os.IsNotExist(err) {
		t.Fatal("Source file should not exist after move")
	}

	// Verify destination file exists with correct content
	dstContent, err = os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(dstContent) != content {
		t.Fatalf("Destination file content mismatch. Expected %q, got %q", content, dstContent)
	}
}

func TestCreateWithBackup(t *testing.T) {
	dir := createTempDir(t)
	originalPath := filepath.Join(dir, "original.txt")

	// Create original file
	createTestFile(t, originalPath, "original content")

	// Test CreateWithBackup
	tempFile, cleanup, commit, err := CreateWithBackup(originalPath)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	defer cleanup()

	// Write to temporary file
	newContent := "new content"
	_, err = tempFile.WriteString(newContent)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Commit changes
	err = commit()
	if err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	// Verify backup file exists
	backupPath := originalPath + ".backup"
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}
	if string(backupContent) != "original content" {
		t.Fatalf("Backup file content mismatch. Expected %q, got %q", "original content", backupContent)
	}

	// Verify original file has new content
	originalContent, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}
	if string(originalContent) != newContent {
		t.Fatalf("Original file content mismatch. Expected %q, got %q", newContent, originalContent)
	}

	// Verify temporary file is removed
	tempFileName := tempFile.Name()
	_, err = os.Stat(tempFileName)
	if !os.IsNotExist(err) {
		t.Fatal("Temporary file should be removed after commit")
	}
}

// Test behavior when original file doesn't exist
func TestCreateWithBackupNoOriginal(t *testing.T) {
	dir := createTempDir(t)
	originalPath := filepath.Join(dir, "new-file.txt")

	// Test CreateWithBackup for non-existing file
	tempFile, cleanup, commit, err := CreateWithBackup(originalPath)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}
	defer cleanup()

	// Write to temporary file
	newContent := "new content"
	_, err = tempFile.WriteString(newContent)
	if err != nil {
		t.Fatalf("Failed to write to temporary file: %v", err)
	}

	// Commit changes
	err = commit()
	if err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	// Verify original file has new content
	originalContent, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}
	if string(originalContent) != newContent {
		t.Fatalf("Original file content mismatch. Expected %q, got %q", newContent, originalContent)
	}

	// Verify no backup file was created
	backupPath := originalPath + ".backup"
	_, err = os.Stat(backupPath)
	if !os.IsNotExist(err) {
		t.Fatal("Backup file should not exist when original file did not exist")
	}
}

// Benchmark Move operation
func BenchmarkMove(b *testing.B) {
	dir := os.TempDir()
	srcPath := filepath.Join(dir, "benchsrc.txt")
	dstPath := filepath.Join(dir, "benchdst.txt")

	// Prepare benchmark by creating source file
	err := os.WriteFile(srcPath, []byte("benchmark content"), 0644)
	if err != nil {
		b.Fatalf("Failed to create source file: %v", err)
	}
	defer os.Remove(srcPath)
	defer os.Remove(dstPath)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Recreate destination path for each iteration
		dstPath := filepath.Join(dir, "benchdst.txt")
		if err := Move(srcPath, dstPath, true); err != nil {
			b.Fatalf("Move failed: %v", err)
		}
	}
}
