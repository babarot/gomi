package atomic

import (
	"errors"
	"fmt"
)

var (
	// ErrDestinationExists indicates that the destination path already exists
	ErrDestinationExists = errors.New("destination already exists")

	// ErrSourceNotFound indicates that the source file does not exist
	ErrSourceNotFound = errors.New("source file not found")

	// ErrCrossDeviceMove indicates a move operation across different devices
	ErrCrossDeviceMove = errors.New("cross-device move operation")

	// ErrInvalidDestination indicates an invalid destination path
	ErrInvalidDestination = errors.New("invalid destination path")

	// ErrTemporaryFileError indicates an error with temporary file operations
	ErrTemporaryFileError = errors.New("temporary file operation failed")

	// ErrTransactionFailed indicates a transaction operation failed
	ErrTransactionFailed = errors.New("transaction operation failed")

	ErrInvalidPath = errors.New("invalid path specified")
)

// MoveError represents an error that occurred during a move operation
type MoveError struct {
	Op  string // Operation being performed
	Src string // Source path
	Dst string // Destination path
	Err error  // Underlying error
}

func (e *MoveError) Error() string {
	return fmt.Sprintf("move operation failed: %s from %q to %q: %v", e.Op, e.Src, e.Dst, e.Err)
}

func (e *MoveError) Unwrap() error {
	return e.Err
}

// NewMoveError creates a new MoveError
func NewMoveError(op, src, dst string, err error) error {
	return &MoveError{
		Op:  op,
		Src: src,
		Dst: dst,
		Err: err,
	}
}

// CleanupError represents an error that occurred during cleanup
type CleanupError struct {
	Path string // Path being cleaned up
	Err  error  // Underlying error
}

func (e *CleanupError) Error() string {
	return fmt.Sprintf("cleanup failed for %q: %v", e.Path, e.Err)
}

func (e *CleanupError) Unwrap() error {
	return e.Err
}

// NewCleanupError creates a new CleanupError
func NewCleanupError(path string, err error) error {
	return &CleanupError{
		Path: path,
		Err:  err,
	}
}

// IsCrossDevice checks if the error indicates a cross-device operation
func IsCrossDevice(err error) bool {
	return errors.Is(err, ErrCrossDeviceMove)
}

// IsDestinationExists checks if the error indicates the destination exists
func IsDestinationExists(err error) bool {
	return errors.Is(err, ErrDestinationExists)
}
