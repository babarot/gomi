package json

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/core/types"
	"github.com/babarot/gomi/internal/storage"
	"github.com/babarot/gomi/internal/utils"
	"github.com/docker/go-units"
	"github.com/gobwas/glob"
	"github.com/k1LoW/duration"
	"github.com/samber/lo"
)

const (
	currentVersion = 1
	tmpDirName     = "tmp"
)

// Store implements the metadata store using JSON files with Two-Phase Commit
type Store struct {
	path    string          // Path to history.json
	tempDir string          // Directory for temporary files
	storage storage.Storage // Storage interface for file operations
	mu      sync.RWMutex    // Mutex for thread safety
	history types.TrashHistory
	config  config.History
}

// NewStore creates a new JSON-based metadata store
func NewStore(path string, tempDir string, storage storage.Storage, cfg config.History) (*Store, error) {
	s := &Store{
		path:    path,
		tempDir: tempDir,
		storage: storage,
		history: types.TrashHistory{Version: currentVersion},
		config:  cfg,
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}

	// Load existing history
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load history: %w", err)
	}

	// Recover any pending transactions
	if err := s.recoverPendingTransactions(); err != nil {
		slog.Warn("failed to recover pending transactions", "error", err)
	}

	return s, nil
}

// PrepareMove prepares a move operation transaction (Phase 1)
func (s *Store) PrepareMove(file types.TrashFile) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Create new transaction
	tx, err := NewTransaction(OperationMove, file, s.tempDir)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// 2. Create temporary history file with new entry
	if err := s.prepareHistoryFile(tx, file); err != nil {
		tx.Cleanup()
		return nil, fmt.Errorf("prepare history file: %w", err)
	}

	// 3. Create backup of the file
	if err := tx.BackupFile(file.From); err != nil {
		tx.Cleanup()
		return nil, fmt.Errorf("backup file: %w", err)
	}

	// 4. Mark transaction as prepared
	if err := tx.Prepare(); err != nil {
		tx.Cleanup()
		return nil, fmt.Errorf("mark transaction prepared: %w", err)
	}

	return tx, nil
}

// CommitMove completes a move operation transaction (Phase 2)
func (s *Store) CommitMove(tx *Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Verify transaction state
	if tx.Metadata.State != StatePrepared {
		return fmt.Errorf("transaction not in prepared state: %s", tx.Metadata.State)
	}

	// 2. Move temporary history file to main history file (atomic operation)
	if err := s.storage.Move(tx.HistoryTempPath, s.path, storage.MoveOptions{
		Atomic: true,
		Force:  true,
	}); err != nil {
		tx.Fail()
		return fmt.Errorf("commit history: %w", err)
	}

	// 3. Remove backup file
	if err := os.Remove(tx.BackupPath); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove backup file", "error", err)
	}

	// 4. Mark transaction as completed
	if err := tx.Complete(); err != nil {
		slog.Error("failed to mark transaction complete", "error", err)
	}

	// 5. Reload history
	if err := s.load(); err != nil {
		slog.Error("failed to reload history after commit", "error", err)
	}

	// 6. Cleanup transaction files
	return tx.Cleanup()
}

// RollbackMove aborts a move operation transaction
func (s *Store) RollbackMove(tx *Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Restore from backup if exists
	if err := tx.RestoreFromBackup(); err != nil {
		slog.Error("failed to restore from backup", "error", err)
	}

	// 2. Mark transaction as rolled back
	if err := tx.RollBack(); err != nil {
		slog.Error("failed to mark transaction as rolled back", "error", err)
	}

	// 3. Cleanup transaction files
	return tx.Cleanup()
}

func (s *Store) prepareHistoryFile(tx *Transaction, file types.TrashFile) error {
	// 1. Copy current history to temporary file
	if err := s.copyHistoryFile(tx.HistoryTempPath); err != nil {
		return fmt.Errorf("copy history: %w", err)
	}

	// 2. Add new entry to temporary history
	newHistory := s.history
	newHistory.Files = append(newHistory.Files, file)

	// 3. Write updated history to temporary file
	if err := s.writeHistoryFile(tx.HistoryTempPath, &newHistory); err != nil {
		return fmt.Errorf("write updated history: %w", err)
	}

	return nil
}

func (s *Store) copyHistoryFile(dst string) error {
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		// If no history exists, create empty one
		return s.writeHistoryFile(dst, &types.TrashHistory{
			Version: currentVersion,
		})
	}

	return s.storage.Copy(s.path, dst, storage.CopyOptions{
		PreserveAll: true,
	})
}

func (s *Store) writeHistoryFile(path string, history *types.TrashHistory) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create history file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(history); err != nil {
		return fmt.Errorf("encode history: %w", err)
	}

	return f.Sync()
}

func (s *Store) load() error {
	f, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(&s.history)
}

// recoverPendingTransactions handles any interrupted transactions
func (s *Store) recoverPendingTransactions() error {
	pending, err := RecoverPending(s.tempDir)
	if err != nil {
		return err
	}

	for _, tx := range pending {
		slog.Info("found pending transaction",
			"id", tx.ID,
			"type", tx.Type,
			"state", tx.Metadata.State,
			"duration", tx.Metadata.Duration())

		switch tx.Metadata.State {
		case StatePrepared:
			// Transaction was prepared but not committed
			if time.Since(tx.Metadata.UpdateTime) > 1*time.Hour {
				// Old transaction - roll back
				if err := s.RollbackMove(tx); err != nil {
					slog.Error("failed to rollback transaction", "error", err)
				}
			}
		case StateInitial, StateFailed:
			// Clean up incomplete transaction
			if err := tx.Cleanup(); err != nil {
				slog.Error("failed to cleanup transaction", "error", err)
			}
		}
	}

	return nil
}

// List returns all files in the history
func (s *Store) List() []types.TrashFile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files := make([]types.TrashFile, len(s.history.Files))
	copy(files, s.history.Files)
	return files
}

// Filter returns filtered files based on configuration criteria
func (s *Store) Filter() []types.TrashFile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy of files to avoid modifying original slice
	files := make([]types.TrashFile, len(s.history.Files))
	copy(files, s.history.Files)

	// Exclude files by name
	files = lo.Reject(files, func(file types.TrashFile, index int) bool {
		return lo.Contains(s.config.Exclude.Files, file.Name)
	})

	// Exclude files by pattern and glob
	files = lo.Reject(files, func(file types.TrashFile, index int) bool {
		// Check patterns (regular expressions)
		for _, pattern := range s.config.Exclude.Patterns {
			if regexp.MustCompile(pattern).MatchString(file.Name) {
				return true
			}
		}

		// Check globs
		for _, g := range s.config.Exclude.Globs {
			if glob.MustCompile(g).Match(file.Name) {
				return true
			}
		}

		return false
	})

	// Exclude files by size
	files = lo.Reject(files, func(file types.TrashFile, index int) bool {
		size, err := utils.DirSize(file.To)
		if err != nil {
			return false // false positive - keep file if size can't be determined
		}

		// Check minimum size
		if s := s.config.Exclude.Size.Min; s != "" {
			min, err := units.FromHumanSize(s)
			if err != nil {
				slog.Error("failed to parse min size", "error", err)
				return false
			}
			if size <= min {
				return true
			}
		}

		// Check maximum size
		if s := s.config.Exclude.Size.Max; s != "" {
			max, err := units.FromHumanSize(s)
			if err != nil {
				slog.Error("failed to parse max size", "error", err)
				return false
			}
			if max <= size {
				return true
			}
		}

		return false
	})

	// Include files by period
	files = lo.Filter(files, func(file types.TrashFile, index int) bool {
		if period := s.config.Include.Period; period > 0 {
			d, err := duration.Parse(fmt.Sprintf("%d days", period))
			if err != nil {
				slog.Error("failed to parse duration", "error", err)
				return false
			}
			if time.Since(file.Timestamp) < d {
				return true
			}
		}
		return false
	})

	return files
}

// Remove removes a file from history
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create temporary history without the file
	newHistory := s.history
	newHistory.Files = make([]types.TrashFile, 0, len(s.history.Files))
	for _, f := range s.history.Files {
		if f.ID != id {
			newHistory.Files = append(newHistory.Files, f)
		}
	}

	// Create temporary file
	tempPath := filepath.Join(s.tempDir, fmt.Sprintf("history_%d.json", time.Now().UnixNano()))
	if err := s.writeHistoryFile(tempPath, &newHistory); err != nil {
		return fmt.Errorf("write temporary history: %w", err)
	}

	// Atomic replace
	if err := s.storage.Move(tempPath, s.path, storage.MoveOptions{
		Atomic: true,
		Force:  true,
	}); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("commit history: %w", err)
	}

	s.history = newHistory
	return nil
}

// PrepareRestore prepares a restore operation (Phase 1)
func (s *Store) PrepareRestore(file types.TrashFile) (*Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Create new transaction
	tx, err := NewTransaction(OperationRestore, file, s.tempDir)
	if err != nil {
		return nil, fmt.Errorf("create transaction: %w", err)
	}

	// 2. Prepare history file with the entry removed
	if err := s.prepareRestoreHistory(tx, file); err != nil {
		tx.Cleanup()
		return nil, fmt.Errorf("prepare history file: %w", err)
	}

	// 3. Create backup of the file in trash
	if err := tx.BackupFile(file.To); err != nil {
		tx.Cleanup()
		return nil, fmt.Errorf("backup file: %w", err)
	}

	// 4. Mark transaction as prepared
	if err := tx.Prepare(); err != nil {
		tx.Cleanup()
		return nil, fmt.Errorf("mark transaction prepared: %w", err)
	}

	return tx, nil
}

func (s *Store) prepareRestoreHistory(tx *Transaction, file types.TrashFile) error {
	// 1. Copy current history to temporary file
	if err := s.copyHistoryFile(tx.HistoryTempPath); err != nil {
		return fmt.Errorf("copy history: %w", err)
	}

	// 2. Create new history without the file being restored
	newHistory := s.history
	newHistory.Files = make([]types.TrashFile, 0, len(s.history.Files))
	for _, f := range s.history.Files {
		if f.ID != file.ID && (!file.IsDir || !strings.HasPrefix(f.From, file.From)) {
			newHistory.Files = append(newHistory.Files, f)
		}
	}

	// 3. Write updated history to temporary file
	if err := s.writeHistoryFile(tx.HistoryTempPath, &newHistory); err != nil {
		return fmt.Errorf("write updated history: %w", err)
	}

	return nil
}

// CommitRestore completes a restore operation (Phase 2)
func (s *Store) CommitRestore(tx *Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Verify transaction state
	if tx.Metadata.State != StatePrepared {
		return fmt.Errorf("transaction not in prepared state: %s", tx.Metadata.State)
	}

	// 2. Move temporary history file to main history file (atomic operation)
	if err := s.storage.Move(tx.HistoryTempPath, s.path, storage.MoveOptions{
		Atomic: true,
		Force:  true,
	}); err != nil {
		tx.Fail()
		return fmt.Errorf("commit history: %w", err)
	}

	// 3. Remove backup file
	if err := os.Remove(tx.BackupPath); err != nil && !os.IsNotExist(err) {
		slog.Error("failed to remove backup file", "error", err)
	}

	// 4. Mark transaction as completed
	if err := tx.Complete(); err != nil {
		slog.Error("failed to mark transaction complete", "error", err)
	}

	// 5. Reload history
	if err := s.load(); err != nil {
		slog.Error("failed to reload history after commit", "error", err)
	}

	// 6. Cleanup transaction files
	return tx.Cleanup()
}

// RollbackRestore aborts a restore operation
func (s *Store) RollbackRestore(tx *Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Restore from backup if exists
	if err := tx.RestoreFromBackup(); err != nil {
		slog.Error("failed to restore from backup", "error", err)
	}

	// 2. Mark transaction as rolled back
	if err := tx.RollBack(); err != nil {
		slog.Error("failed to mark transaction as rolled back", "error", err)
	}

	// 3. Cleanup transaction files
	return tx.Cleanup()
}
