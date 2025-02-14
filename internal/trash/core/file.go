package core

import (
	"io/fs"
	"os"
	"time"
)

// File represents a file in trash
type File struct {
	// Name is the original base name of the file
	Name string

	// OriginalPath is the absolute path where the file was located
	OriginalPath string

	// TrashPath is the absolute path where the file is stored in trash
	TrashPath string

	// DeletedAt is when the file was moved to trash
	DeletedAt time.Time

	// Size is the size of the file in bytes
	Size int64

	// IsDir indicates if this is a directory
	IsDir bool

	// FileMode is the original mode of the file
	FileMode fs.FileMode

	// storage is a reference to the Storage implementation that manages this file
	// This field is used internally by implementations and is not exported
	storage Storage
}

// Exists checks if the file still exists in the trash
func (f *File) Exists() bool {
	_, err := os.Stat(f.TrashPath)
	return err == nil
}

// RequiresAdmin returns true if administrator privileges are required
// to restore or remove this file
func (f *File) RequiresAdmin() bool {
	info, err := os.Stat(f.TrashPath)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0200 == 0
}

// SetStorage sets the storage reference for this file
// This is used internally by Storage implementations
func (f *File) SetStorage(s Storage) {
	f.storage = s
}

// GetStorage returns the storage reference for this file
// This is used internally by Storage implementations
func (f *File) GetStorage() Storage {
	return f.storage
}
