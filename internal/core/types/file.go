package types

import "time"

// TrashFile represents a file in the trash
type TrashFile struct {
	Name      string    `json:"name"`
	ID        string    `json:"id"`
	RunID     string    `json:"group_id"` // for backward compatibility
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
	IsDir     bool      `json:"is_dir"`
}

// TrashHistory represents the history file structure
type TrashHistory struct {
	Version int         `json:"version"`
	Files   []TrashFile `json:"files"`
}

// FileOperation represents the operation being performed on a file
type FileOperation struct {
	Type      string    `json:"type"`
	File      TrashFile `json:"file"`
	Timestamp time.Time `json:"timestamp"`
}

// ValidationError represents a file validation error
type ValidationError struct {
	Path    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(path, message string) *ValidationError {
	return &ValidationError{
		Path:    path,
		Message: message,
	}
}
