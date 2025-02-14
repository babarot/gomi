package legacy

/*
TODO: merge this file into history/

const (
	// Version of the history file format
	historyVersion = 1
)

// History represents the legacy trash history
type History struct {
	// Version of the history file format
	Version int `json:"version"`

	// List of files in the trash
	Files []*File `json:"files"`
}

// File represents a file in the legacy trash history
type File struct {
	// Original base name of the file
	Name string `json:"name"`

	// Unique identifier for the file
	ID string `json:"id"`

	// Group ID for batch operations (reusing existing field name for compatibility)
	RunID string `json:"group_id"`

	// Original absolute path of the file
	From string `json:"from"`

	// Path in the trash directory
	To string `json:"to"`

	// When the file was trashed
	Timestamp time.Time `json:"timestamp"`
}

// NewHistory creates a new History instance
func NewHistory() *History {
	return &History{
		Version: historyVersion,
		Files:   make([]*File, 0),
	}
}

// Add adds a file to the history
func (h *History) Add(file *File) {
	h.Files = append(h.Files, file)
}

// Remove removes a file from the history by its trash name
func (h *History) Remove(trashName string) {
	var filtered []*File
	for _, f := range h.Files {
		if filepath.Base(f.To) != trashName {
			filtered = append(filtered, f)
		}
	}
	h.Files = filtered
}

// RemoveByPath removes a file from the history by its original path
func (h *History) RemoveByPath(path string) {
	var filtered []*File
	for _, f := range h.Files {
		if f.From != path {
			filtered = append(filtered, f)
		}
	}
	h.Files = filtered
}

// FindByID finds a file in the history by its ID
func (h *History) FindByID(id string) *File {
	for _, f := range h.Files {
		if f.ID == id {
			return f
		}
	}
	return nil
}

// FindByPath finds a file in the history by its original path
func (h *History) FindByPath(path string) *File {
	for _, f := range h.Files {
		if f.From == path {
			return f
		}
	}
	return nil
}

// Backup creates a backup of the history file
func (h *History) Backup(historyPath string) error {
	backupPath := historyPath + ".backup"
	src, err := os.Open(historyPath)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	if err := dst.Sync(); err != nil {
		return fmt.Errorf("failed to sync backup file: %w", err)
	}

	return nil
}

// RestoreFromBackup restores the history from a backup file
func (h *History) RestoreFromBackup(historyPath string) error {
	backupPath := historyPath + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	src, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(historyPath)
	if err != nil {
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer dst.Close()

	if _, err := dst.ReadFrom(src); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	if err := dst.Sync(); err != nil {
		return fmt.Errorf("failed to sync history file: %w", err)
	}

	return nil
}

// Clean removes entries for files that no longer exist in the trash
func (h *History) Clean() {
	var filtered []*File
	for _, f := range h.Files {
		if _, err := os.Stat(f.To); err == nil {
			filtered = append(filtered, f)
		}
	}
	h.Files = filtered
}

// Exists checks if a file exists in the history
func (h *History) Exists(path string) bool {
	return h.FindByPath(path) != nil
}

// Count returns the number of files in the history
func (h *History) Count() int {
	return len(h.Files)
}
*/
