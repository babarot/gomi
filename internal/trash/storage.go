// Package trash provides the core functionality for trash management
package trash

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// StorageType represents the type of trash storage
type StorageType int

const (
	// StorageTypeXDG represents XDG-compliant trash storage
	StorageTypeXDG StorageType = iota

	// StorageTypeLegacy represents legacy (.gomi) trash storage
	StorageTypeLegacy
)

func (t StorageType) String() string {
	switch t {
	case StorageTypeXDG:
		return "xdg"
	case StorageTypeLegacy:
		return "legacy"
	default:
		return "unknown"
	}
}

// StorageLocation represents where the trash storage is located
type StorageLocation int

const (
	// LocationHome indicates home directory storage
	LocationHome StorageLocation = iota

	// LocationExternal indicates external device storage
	LocationExternal
)

// StorageInfo provides information about a trash storage
type StorageInfo struct {
	// Location indicates whether this is a home or external storage
	Location StorageLocation

	// Trashes is the root directory of this storage (e.g., ~/.local/share/Trash)
	Trashes []string

	// Available indicates whether this storage is currently available
	// (e.g., external storage might become unavailable)
	Available bool

	// Type indicates the storage implementation type
	Type StorageType
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

	// MountRoot is the root path of the mount point containing this trash
	// This is used to resolve relative paths in .trashinfo files
	MountRoot string

	// storage is a reference to the Storage implementation that manages this file
	storage Storage
}

func (f *File) GetName() string {
	return f.Name
}

func (f *File) GetPath() string {
	return f.TrashPath
}

func (f *File) GetDeletedAt() time.Time {
	return f.DeletedAt
}

// Exists checks if the file still exists in the trash
func (f *File) Exists() bool {
	_, err := os.Stat(f.TrashPath)
	exists := err == nil
	return exists
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
func (f *File) SetStorage(s Storage) {
	f.storage = s
}

// GetStorage returns the storage reference for this file
func (f *File) GetStorage() Storage {
	return f.storage
}

// GetOriginalPath returns the absolute original path of the file
// It handles both absolute and relative paths from .trashinfo files
func (f *File) GetOriginalPath() string {
	// If original path is already absolute, return as is
	if filepath.IsAbs(f.OriginalPath) {
		return f.OriginalPath
	}

	// For relative paths, join with mount root
	if f.MountRoot != "" {
		return filepath.Join(f.MountRoot, f.OriginalPath)
	}

	// Fallback to original path if no mount root is available
	return f.OriginalPath
}

// GetRelativePath returns the path relative to the mount root
// This is used when saving .trashinfo files
func (f *File) GetRelativePath() string {
	if f.MountRoot == "" || !filepath.IsAbs(f.OriginalPath) {
		return f.OriginalPath
	}

	rel, err := filepath.Rel(f.MountRoot, f.OriginalPath)
	if err != nil {
		return f.OriginalPath
	}
	return rel
}

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

// StorageConstructor is a function type for creating new Storage instances
type StorageConstructor func(Config) (Storage, error)

// IsExistLegacy checks if legacy storage exists
func IsExistLegacy() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("failed to get home directory: %w", err)
	}

	legacyPath := filepath.Join(home, ".gomi")
	if fi, err := os.Stat(legacyPath); err == nil && fi.IsDir() {
		return true, nil
	}

	return false, nil
}
