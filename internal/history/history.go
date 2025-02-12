package history

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/core/types"
	"github.com/babarot/gomi/internal/history/json"
	"github.com/babarot/gomi/internal/storage"
	"github.com/google/uuid"
)

// History manages the trash history
type History struct {
	store    *json.Store
	trashDir string
	tempDir  string
	runID    string
}

// New creates a new History instance
func New(trashDir string, config config.History) (*History, error) {
	if trashDir == "" {
		trashDir = filepath.Join(os.Getenv("HOME"), ".gomi")
	}

	// Create required directories
	tempDir := filepath.Join(trashDir, "temp")
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}

	// Initialize storage
	localStorage := storage.NewLocalStorage(tempDir)

	// Initialize store
	store, err := json.NewStore(
		filepath.Join(trashDir, "history.json"),
		tempDir,
		localStorage,
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize store: %w", err)
	}

	return &History{
		store:    store,
		tempDir:  tempDir,
		trashDir: trashDir,
		runID:    uuid.New().String(),
	}, nil
}

// FileInfo creates a new file entry
func (h *History) FileInfo(runID string, path string) (types.TrashFile, error) {
	name := filepath.Base(path)
	from, err := filepath.Abs(path)
	if err != nil {
		return types.TrashFile{}, err
	}

	id := uuid.New().String()
	now := time.Now()

	return types.TrashFile{
		Name:  name,
		ID:    id,
		RunID: runID,
		From:  from,
		To: filepath.Join(
			// h.tempDir,
			h.trashDir,
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
			runID,
			fmt.Sprintf("%s.%s", name, id),
		),
		Timestamp: now,
	}, nil
}

// PrepareMove prepares a move operation
func (h *History) PrepareMove(file types.TrashFile) (*json.Transaction, error) {
	return h.store.PrepareMove(file)
}

// CommitMove commits a move operation
func (h *History) CommitMove(tx *json.Transaction) error {
	return h.store.CommitMove(tx)
}

// RollbackMove rolls back a move operation
func (h *History) RollbackMove(tx *json.Transaction) error {
	return h.store.RollbackMove(tx)
}

// PrepareRestore prepares a restore operation
func (h *History) PrepareRestore(file types.TrashFile) (*json.Transaction, error) {
	return h.store.PrepareRestore(file)
}

// CommitRestore commits a restore operation
func (h *History) CommitRestore(tx *json.Transaction) error {
	return h.store.CommitRestore(tx)
}

// RollbackRestore rolls back a restore operation
func (h *History) RollbackRestore(tx *json.Transaction) error {
	return h.store.RollbackRestore(tx)
}

// List returns all files in the history
func (h *History) List() []types.TrashFile {
	return h.store.List()
}

// Filter returns filtered files based on criteria
func (h *History) Filter() []types.TrashFile {
	return h.store.Filter()
}

// Remove removes a file from history
func (h *History) Remove(file types.TrashFile) error {
	return h.store.Remove(file.ID)
}
