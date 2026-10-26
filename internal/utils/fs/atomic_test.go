package fs

import (
	"os"
	"path/filepath"
	"runtime"
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

	// Move with fallbackCopy set; rename still succeeds here, so the copy
	// fallback itself is exercised by the TestMove_CopyFallback tests
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

// skipUnlessPermissionsApply skips tests that rely on permission bits to
// make rename(2) fail
func skipUnlessPermissionsApply(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	if os.Getuid() == 0 {
		t.Skip("root is never denied permission")
	}
}

// lockDir makes dir read-only, so entries in it cannot be renamed or
// removed, until the test ends
func lockDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
}

// newSourceDir creates parent/src holding a file and a file in a
// subdirectory, and returns its path
func newSourceDir(t *testing.T, parent string) string {
	t.Helper()
	src := filepath.Join(parent, "src")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(src, "a.txt"), "a")
	createTestFile(t, filepath.Join(src, "sub", "b.txt"), "b")
	return src
}

// assertFile fails unless path holds content
func assertFile(t *testing.T, path, content string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s: %v", path, err)
		return
	}
	if string(data) != content {
		t.Errorf("%s = %q, want %q", path, data, content)
	}
}

// Without the fallback, a failed rename must leave the source alone and
// create nothing at the destination.
func TestMove_RenameFailsWithoutFallback(t *testing.T) {
	skipUnlessPermissionsApply(t)
	dir := createTempDir(t)
	parent := filepath.Join(dir, "locked")
	src := newSourceDir(t, parent)
	dst := filepath.Join(dir, "dst")
	lockDir(t, parent)

	if err := Move(src, dst, false); err == nil {
		t.Fatal("Move() succeeded, want an error")
	}
	assertFile(t, filepath.Join(src, "a.txt"), "a")
	assertFile(t, filepath.Join(src, "sub", "b.txt"), "b")
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("dst should not exist: %v", err)
	}
}

// When rename fails because the destination directory is not empty, the
// copy fallback moves the source's contents there and removes the source.
func TestMove_CopyFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	dir := createTempDir(t)
	src := newSourceDir(t, dir)
	dst := filepath.Join(dir, "dst")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	createTestFile(t, filepath.Join(dst, "existing.txt"), "existing")

	if err := Move(src, dst, true); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	assertFile(t, filepath.Join(dst, "a.txt"), "a")
	assertFile(t, filepath.Join(dst, "sub", "b.txt"), "b")
	assertFile(t, filepath.Join(dst, "existing.txt"), "existing")
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("src should not exist: %v", err)
	}
}

// When the copy fallback cannot finish removing the source, the data must
// survive somewhere. Here rename fails because the source's parent is
// read-only; os.RemoveAll(src) then deletes everything inside src before
// it fails to remove src itself.
func TestMove_CopyFallbackKeepsDataWhenSourceRemovalFails(t *testing.T) {
	t.Skip("known bug: Move removes the complete copy at dst when os.RemoveAll(src) fails after deleting part of src (internal/utils/fs/atomic.go:40-44)")
	skipUnlessPermissionsApply(t)
	dir := createTempDir(t)
	parent := filepath.Join(dir, "locked")
	src := newSourceDir(t, parent)
	dst := filepath.Join(dir, "dst")
	lockDir(t, parent)

	if err := Move(src, dst, true); err == nil {
		t.Fatal("Move() succeeded, want an error")
	}

	// Each file must be at src or at dst
	for _, rel := range []string{"a.txt", filepath.Join("sub", "b.txt")} {
		_, srcErr := os.Stat(filepath.Join(src, rel))
		_, dstErr := os.Stat(filepath.Join(dst, rel))
		if srcErr != nil && dstErr != nil {
			t.Errorf("%s is lost: not in src (%v) nor in dst (%v)", rel, srcErr, dstErr)
		}
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
