package core

import "errors"

// Common errors that can be returned by Storage implementations
var (
	ErrNotFound         = errors.New("file not found in trash")
	ErrInvalidStorage   = errors.New("invalid storage")
	ErrCrossDevice      = errors.New("cross-device operation not supported")
	ErrStorageNotReady  = errors.New("storage is not ready")
	ErrPermissionDenied = errors.New("permission denied")
	ErrFileExists       = errors.New("file already exists")
)

// StorageError wraps an error with additional context about the storage operation
type StorageError struct {
	Op   string // Operation that failed (e.g., "put", "restore", "remove")
	Path string // Path of the file that caused the error
	Err  error  // The underlying error
}

func (e *StorageError) Error() string {
	if e.Path == "" {
		return e.Op + ": " + e.Err.Error()
	}
	return e.Op + " " + e.Path + ": " + e.Err.Error()
}

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
