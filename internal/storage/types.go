package storage

import (
	"errors"
	"io"
	"os"
	"time"
)

// Common errors for storage operations
var (
	ErrInvalidPath      = errors.New("invalid path specified")
	ErrFileExists       = errors.New("file already exists")
	ErrNotFound         = errors.New("file not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrIsDirectory      = errors.New("path is a directory")
	ErrNotDirectory     = errors.New("path is not a directory")
)

// FileInfo represents information about a file in storage
type FileInfo struct {
	Path     string      // File path
	Size     int64       // File size in bytes
	ModTime  time.Time   // Last modification time
	Mode     os.FileMode // File mode and permission
	IsDir    bool        // Whether this is a directory
	DeviceID uint64      // Device ID where the file is stored
}

// DirectoryEntry represents an entry in a directory
type DirectoryEntry struct {
	Name     string      // Base name of the entry
	Path     string      // Full path to the entry
	Size     int64       // Size in bytes
	ModTime  time.Time   // Modification time
	Mode     os.FileMode // File mode
	IsDir    bool        // Whether this is a directory
	Children []string    // Child paths (for directories)
}

// CopyOptions specifies options for copy operations
type CopyOptions struct {
	Overwrite    bool                         // Whether to overwrite existing files
	PreserveAll  bool                         // Preserve all attributes
	Recursive    bool                         // Copy directories recursively
	FollowLinks  bool                         // Follow symbolic links
	BufferSize   int                          // Buffer size for copying
	Progress     io.Writer                    // Writer for progress updates
	OnFileStart  func(path string) error      // Called before copying each file
	OnFileFinish func(path string, err error) // Called after copying each file
	OnDirStart   func(path string) error      // Called before processing each directory
	OnDirFinish  func(path string, err error) // Called after processing each directory
}

// MoveOptions specifies options for move operations
type MoveOptions struct {
	AllowCrossDevice bool                         // Allow cross-device moves
	Force            bool                         // Force operation even if destination exists
	Atomic           bool                         // Ensure atomic operation
	Progress         io.Writer                    // Writer for progress updates
	OnStart          func(path string) error      // Called before moving
	OnFinish         func(path string, err error) // Called after moving
}

// Storage defines the interface for storage operations
type Storage interface {
	// Info returns file information
	Info(path string) (*FileInfo, error)

	// List returns directory entries
	List(path string) ([]DirectoryEntry, error)

	// Copy copies a file or directory with the specified options
	Copy(src, dst string, opts CopyOptions) error

	// Move moves a file or directory with the specified options
	Move(src, dst string, opts MoveOptions) error

	// Remove removes the specified file or directory
	Remove(path string) error

	// CreateTemp creates a temporary file in the specified directory
	CreateTemp(dir, pattern string) (string, error)

	// IsSameDevice checks if two paths are on the same device
	IsSameDevice(path1, path2 string) (bool, error)

	// Walk walks the file tree rooted at root
	Walk(root string, fn WalkFunc) error
}

// WalkFunc is called for each file or directory visited by Walk.
// The path argument contains the argument to Walk as a prefix.
// If there was a problem walking to the file or directory named by path,
// the incoming error will describe the problem and the function can decide
// how to handle that error.
// If an error is returned, processing stops.
type WalkFunc func(path string, info DirectoryEntry, err error) error

// PathValidator provides path validation functionality
type PathValidator interface {
	Validate(path string) error
	IsAbsolute(path string) bool
}

// ProgressWriter provides progress reporting functionality
type ProgressWriter interface {
	Write(p []byte) (n int, err error)
	Progress() float64
	SetTotal(total int64)
	AddTotal(delta int64)
}

// Result represents the result of a storage operation
type Result struct {
	Success     bool      // Whether the operation was successful
	Path        string    // Affected path
	Error       error     // Any error that occurred
	Timestamp   time.Time // When the operation completed
	BytesCopied int64     // Number of bytes copied
	ItemsCopied int       // Number of items processed
}
