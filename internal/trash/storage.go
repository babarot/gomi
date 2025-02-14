package trash

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// Common errors that can be returned by Storage implementations
var (
	ErrNotFound         = errors.New("file not found in trash")
	ErrInvalidStorage   = errors.New("invalid storage")
	ErrCrossDevice      = errors.New("cross-device operation not supported")
	ErrStorageNotReady  = errors.New("storage is not ready")
	ErrPermissionDenied = errors.New("permission denied")
	ErrFileExists       = errors.New("file already exists")
)

// Storage defines the interface for different trash implementations
type Storage interface {
	// Put moves the file at src path to trash
	Put(src string) error

	// Restore restores the given file from trash to its original location
	// If dst is specified, the file will be restored to that location instead
	Restore(file *File, dst string) error

	// Remove permanently removes the file from trash
	Remove(file *File) error

	// List returns a list of all files in trash
	List() ([]*File, error)

	// Info returns detailed information about the storage
	Info() *StorageInfo
}

// StorageLocation represents where the trash storage is located
type StorageLocation int

const (
	LocationHome StorageLocation = iota
	LocationExternal
)

// StorageInfo provides information about a trash storage
type StorageInfo struct {
	// Location indicates whether this is a home or external storage
	Location StorageLocation

	// Root is the root directory of this storage (e.g., ~/.local/share/Trash)
	Root string

	// Available indicates whether this storage is currently available
	// (e.g., external storage might become unavailable)
	Available bool
}

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
	storage Storage
}

// setStorage sets the storage reference for this file
// This is used internally by Storage implementations
func (f *File) setStorage(s Storage) {
	f.storage = s
}

// Restore restores the file to its original location or specified destination
func (f *File) Restore(dst string) error {
	if f.storage == nil {
		return fmt.Errorf("cannot restore: %w", ErrInvalidStorage)
	}
	return f.storage.Restore(f, dst)
}

// Remove permanently removes the file from trash
func (f *File) Remove() error {
	if f.storage == nil {
		return fmt.Errorf("cannot remove: %w", ErrInvalidStorage)
	}
	return f.storage.Remove(f)
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
