package core

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

	Type StorageType
}
