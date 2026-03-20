package trash

import (
	"errors"
	"fmt"
	"testing"
)

func TestStorageError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *StorageError
		want string
	}{
		{
			name: "with path",
			err:  &StorageError{Op: "put", Path: "/tmp/file", Err: ErrNotFound},
			want: "put /tmp/file: file not found in trash",
		},
		{
			name: "without path",
			err:  &StorageError{Op: "list", Path: "", Err: ErrStorageNotReady},
			want: "list: storage is not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStorageError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("inner error")
	se := &StorageError{Op: "put", Path: "/tmp/file", Err: inner}
	if got := se.Unwrap(); got != inner {
		t.Errorf("Unwrap() = %v, want %v", got, inner)
	}
}

func TestNewStorageError(t *testing.T) {
	err := NewStorageError("restore", "/tmp/file", ErrNotFound)
	var se *StorageError
	if !errors.As(err, &se) {
		t.Fatal("expected *StorageError")
	}
	if se.Op != "restore" || se.Path != "/tmp/file" || se.Err != ErrNotFound {
		t.Errorf("unexpected fields: %+v", se)
	}
}

func TestErrorCheckers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		{"IsNotFound true", ErrNotFound, IsNotFound, true},
		{"IsNotFound false", ErrInvalidStorage, IsNotFound, false},
		{"IsNotFound wrapped", NewStorageError("op", "", ErrNotFound), IsNotFound, true},
		{"IsInvalidStorage true", ErrInvalidStorage, IsInvalidStorage, true},
		{"IsInvalidStorage false", ErrNotFound, IsInvalidStorage, false},
		{"IsCrossDevice true", ErrCrossDevice, IsCrossDevice, true},
		{"IsCrossDevice false", ErrNotFound, IsCrossDevice, false},
		{"IsStorageNotReady true", ErrStorageNotReady, IsStorageNotReady, true},
		{"IsStorageNotReady false", ErrNotFound, IsStorageNotReady, false},
		{"IsPermissionDenied true", ErrPermissionDenied, IsPermissionDenied, true},
		{"IsPermissionDenied false", ErrNotFound, IsPermissionDenied, false},
		{"IsFileExists true", ErrFileExists, IsFileExists, true},
		{"IsFileExists false", ErrNotFound, IsFileExists, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checker(tt.err); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
