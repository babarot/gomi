package trash

import "errors"

// Common errors that can be returned by Storage implementations
var (
	// ErrNotFound is returned when a file cannot be found in trash
	ErrNotFound = errors.New("file not found in trash")

	// ErrInvalidStorage is returned when a storage operation is attempted with an invalid storage
	ErrInvalidStorage = errors.New("invalid storage")

	// ErrCrossDevice is returned when attempting operations across different devices
	ErrCrossDevice = errors.New("cross-device operation not supported")

	// ErrStorageNotReady is returned when the storage is not in a usable state
	ErrStorageNotReady = errors.New("storage is not ready")

	// ErrPermissionDenied is returned when permission is denied for an operation
	ErrPermissionDenied = errors.New("permission denied")

	// ErrFileExists is returned when a file already exists at the target location
	ErrFileExists = errors.New("file already exists")
)

// StorageError wraps an error with additional context about the storage operation
type StorageError struct {
	// Op is the operation that failed (e.g., "put", "restore", "remove")
	Op string

	// Path is the path of the file that caused the error
	Path string

	// Err is the underlying error
	Err error
}

// Error implements the error interface
func (e *StorageError) Error() string {
	if e.Path == "" {
		return e.Op + ": " + e.Err.Error()
	}
	return e.Op + " " + e.Path + ": " + e.Err.Error()
}

// Unwrap returns the underlying error
func (e *StorageError) Unwrap() error {
	return e.Err
}

// NewStorageError creates a new StorageError
func NewStorageError(op, path string, err error) error {
	return &StorageError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

// IsNotFound returns true if the error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsInvalidStorage returns true if the error is ErrInvalidStorage
func IsInvalidStorage(err error) bool {
	return errors.Is(err, ErrInvalidStorage)
}

// IsCrossDevice returns true if the error is ErrCrossDevice
func IsCrossDevice(err error) bool {
	return errors.Is(err, ErrCrossDevice)
}

// IsStorageNotReady returns true if the error is ErrStorageNotReady
func IsStorageNotReady(err error) bool {
	return errors.Is(err, ErrStorageNotReady)
}

// IsPermissionDenied returns true if the error is ErrPermissionDenied
func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

// IsFileExists returns true if the error is ErrFileExists
func IsFileExists(err error) bool {
	return errors.Is(err, ErrFileExists)
}
