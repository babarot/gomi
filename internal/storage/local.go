package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/babarot/gomi/internal/core/atomic"
)

// LocalStorage implements Storage interface for local filesystem
type LocalStorage struct {
	tempDir string
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(tempDir string) *LocalStorage {
	return &LocalStorage{
		tempDir: tempDir,
	}
}

// Info returns file information
func (s *LocalStorage) Info(path string) (*FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	sys, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("unable to get system info")
	}

	return &FileInfo{
		Path:     path,
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		Mode:     info.Mode(),
		IsDir:    info.IsDir(),
		DeviceID: uint64(sys.Dev),
	}, nil
}

// List returns directory entries
func (s *LocalStorage) List(path string) ([]DirectoryEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]DirectoryEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		dirEntry := DirectoryEntry{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   entry.IsDir(),
		}

		if entry.IsDir() {
			// Get child paths for directories
			children, err := filepath.Glob(filepath.Join(dirEntry.Path, "*"))
			if err == nil {
				dirEntry.Children = children
			}
		}

		result = append(result, dirEntry)
	}

	return result, nil
}

// Move moves a file or directory with the specified options
func (s *LocalStorage) Move(src, dst string, opts MoveOptions) error {
	if err := s.validatePaths(src, dst); err != nil {
		return err
	}

	// Call start callback if provided
	if opts.OnStart != nil {
		if err := opts.OnStart(src); err != nil {
			return err
		}
	}

	// Defer finish callback
	defer func() {
		if opts.OnFinish != nil {
			opts.OnFinish(src, nil)
		}
	}()

	// Use atomic move for atomic operations
	if opts.Atomic {
		return atomic.Move(src, dst, atomic.MoveOptions{
			AllowCrossDev: opts.AllowCrossDevice,
			Force:         opts.Force,
		})
	}

	// Handle directory case
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return s.moveDirectory(src, dst, opts)
	}

	return s.moveFile(src, dst, opts)
}

// moveFile handles single file move
func (s *LocalStorage) moveFile(src, dst string, opts MoveOptions) error {
	// Try direct rename first
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !opts.AllowCrossDevice {
		return err
	}

	// Fall back to copy and delete
	return s.copyAndDelete(src, dst, opts)
}

// moveDirectory handles directory move
func (s *LocalStorage) moveDirectory(src, dst string, opts MoveOptions) error {
	// Try direct rename first
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !opts.AllowCrossDevice {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	// Copy contents
	copyOpts := CopyOptions{
		Recursive:   true,
		PreserveAll: true,
		Progress:    opts.Progress,
	}
	if err := s.Copy(src, dst, copyOpts); err != nil {
		os.RemoveAll(dst)
		return err
	}

	// Remove source directory
	return os.RemoveAll(src)
}

// Copy copies a file or directory with the specified options
func (s *LocalStorage) Copy(src, dst string, opts CopyOptions) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if !opts.Recursive {
			return ErrIsDirectory
		}
		return s.copyDirectory(src, dst, opts)
	}

	return s.copyFile(src, dst, opts)
}

// copyFile handles single file copy
func (s *LocalStorage) copyFile(src, dst string, opts CopyOptions) error {
	if opts.OnFileStart != nil {
		if err := opts.OnFileStart(src); err != nil {
			return err
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	flags := os.O_WRONLY | os.O_CREATE
	if !opts.Overwrite {
		flags |= os.O_EXCL
	}

	dstFile, err := os.OpenFile(dst, flags, 0600)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	var writer io.Writer = dstFile
	if opts.Progress != nil {
		writer = io.MultiWriter(dstFile, opts.Progress)
	}

	if _, err := io.Copy(writer, srcFile); err != nil {
		return err
	}

	if err := dstFile.Sync(); err != nil {
		return err
	}

	if opts.PreserveAll {
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if err := os.Chmod(dst, info.Mode()); err != nil {
			return err
		}
		if err := os.Chtimes(dst, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
	}

	if opts.OnFileFinish != nil {
		opts.OnFileFinish(src, nil)
	}

	return nil
}

// copyDirectory handles directory copy
func (s *LocalStorage) copyDirectory(src, dst string, opts CopyOptions) error {
	if opts.OnDirStart != nil {
		if err := opts.OnDirStart(src); err != nil {
			return err
		}
	}

	// Create destination directory
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	// Read directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := s.copyDirectory(srcPath, dstPath, opts); err != nil {
				return err
			}
		} else {
			if err := s.copyFile(srcPath, dstPath, opts); err != nil {
				return err
			}
		}
	}

	if opts.OnDirFinish != nil {
		opts.OnDirFinish(src, nil)
	}

	return nil
}

// Remove removes the specified file or directory
func (s *LocalStorage) Remove(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(path)
	}
	return os.Remove(path)
}

// CreateTemp creates a temporary file
func (s *LocalStorage) CreateTemp(dir, pattern string) (string, error) {
	if dir == "" {
		dir = s.tempDir
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}

	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return f.Name(), nil
}

// IsSameDevice checks if two paths are on the same device
func (s *LocalStorage) IsSameDevice(path1, path2 string) (bool, error) {
	info1, err := s.Info(path1)
	if err != nil {
		return false, err
	}

	info2, err := s.Info(path2)
	if err != nil {
		return false, err
	}

	return info1.DeviceID == info2.DeviceID, nil
}

func (s *LocalStorage) Walk(root string, fn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fn(path, DirectoryEntry{}, err)
		}

		entry := DirectoryEntry{
			Name:    info.Name(),
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   info.IsDir(),
		}

		return fn(path, entry, nil)
	})
}

// validatePaths performs basic path validation
func (s *LocalStorage) validatePaths(src, dst string) error {
	if src == "" || dst == "" {
		return ErrInvalidPath
	}

	if !filepath.IsAbs(src) || !filepath.IsAbs(dst) {
		return ErrInvalidPath
	}

	return nil
}

// copyAndDelete implements copy and delete for cross-device moves
func (s *LocalStorage) copyAndDelete(src, dst string, opts MoveOptions) error {
	// Copy with progress if writer is provided
	copyOpts := CopyOptions{
		Overwrite:   opts.Force,
		PreserveAll: true,
		Progress:    opts.Progress,
		OnFileStart: func(path string) error {
			if opts.OnStart != nil {
				return opts.OnStart(path)
			}
			return nil
		},
		OnFileFinish: func(path string, err error) {
			if opts.OnFinish != nil {
				opts.OnFinish(path, err)
			}
		},
	}

	if err := s.Copy(src, dst, copyOpts); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// Remove source after successful copy
	if err := os.Remove(src); err != nil {
		// Try to cleanup destination on failure
		os.Remove(dst)
		return fmt.Errorf("remove source failed: %w", err)
	}

	return nil
}
