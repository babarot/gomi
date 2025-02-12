package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/babarot/gomi/internal/core/types"
	"github.com/google/uuid"
	cp "github.com/otiai10/copy"
)

// Transaction represents a file operation transaction
type Transaction struct {
	ID              string              `json:"id"`
	Type            OperationType       `json:"type"`
	File            types.TrashFile     `json:"file"`
	Metadata        TransactionMetadata `json:"metadata"`
	TempPath        string              `json:"temp_path"`         // Path for transaction file
	HistoryTempPath string              `json:"history_temp_path"` // Path for temporary history file
	BackupPath      string              `json:"backup_path"`       // Path for backup of original file
}

// OperationType defines the type of operation being performed
type OperationType string

const (
	OperationMove    OperationType = "move"
	OperationRestore OperationType = "restore"
)

// NewTransaction creates a new transaction with the specified type and file
func NewTransaction(opType OperationType, file types.TrashFile, tempDir string) (*Transaction, error) {
	id := uuid.New().String()
	tx := &Transaction{
		ID:   id,
		Type: opType,
		File: file,
		Metadata: NewTransactionMetadata(fmt.Sprintf("%s operation for %s",
			opType, file.Name)),
		TempPath:        filepath.Join(tempDir, fmt.Sprintf("tx_%s.json", id)),
		HistoryTempPath: filepath.Join(tempDir, fmt.Sprintf("history_%s.json", id)),
		BackupPath:      filepath.Join(tempDir, fmt.Sprintf("backup_%s", id)),
	}

	// Ensure temp directory exists
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}

	// Save initial transaction state
	if err := tx.save(); err != nil {
		return nil, fmt.Errorf("save initial transaction: %w", err)
	}

	return tx, nil
}

// save writes the transaction to its temporary file
func (tx *Transaction) save() error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(tx.TempPath), 0700); err != nil {
		return fmt.Errorf("create transaction directory: %w", err)
	}

	f, err := os.OpenFile(tx.TempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create transaction file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(tx); err != nil {
		return fmt.Errorf("encode transaction: %w", err)
	}

	return f.Sync()
}

// load reads the transaction from its temporary file
func (tx *Transaction) load() error {
	f, err := os.Open(tx.TempPath)
	if err != nil {
		return fmt.Errorf("open transaction file: %w", err)
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(tx)
}

// Prepare moves the transaction to the prepared state
func (tx *Transaction) Prepare() error {
	if err := tx.Metadata.Transition(StatePrepared); err != nil {
		return fmt.Errorf("transition to prepared state: %w", err)
	}
	return tx.save()
}

// Complete marks the transaction as completed
func (tx *Transaction) Complete() error {
	if err := tx.Metadata.Transition(StateCommitted); err != nil {
		return fmt.Errorf("transition to committed state: %w", err)
	}
	return tx.save()
}

// Fail marks the transaction as failed
func (tx *Transaction) Fail() error {
	if err := tx.Metadata.SetError(fmt.Errorf("transaction failed")); err != nil {
		return fmt.Errorf("set failed state: %w", err)
	}
	return tx.save()
}

// RollBack marks the transaction as rolled back
func (tx *Transaction) RollBack() error {
	if err := tx.Metadata.Transition(StateRolledBack); err != nil {
		return fmt.Errorf("transition to rolled back state: %w", err)
	}
	return tx.save()
}

// Cleanup removes all temporary files associated with the transaction
func (tx *Transaction) Cleanup() error {
	var errs []error

	// Remove transaction file
	if err := os.Remove(tx.TempPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("remove transaction file: %w", err))
	}

	// Remove temporary history file
	if err := os.Remove(tx.HistoryTempPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("remove history file: %w", err))
	}

	// Remove backup file if it exists
	if err := os.RemoveAll(tx.BackupPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Errorf("remove backup file: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

// IsActive checks if the transaction is still in progress
func (tx *Transaction) IsActive() bool {
	return !tx.Metadata.State.IsTerminal()
}

// RecoverPending scans for and loads any pending transactions
func RecoverPending(tempDir string) ([]*Transaction, error) {
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("read temp directory: %w", err)
	}

	var pending []*Transaction
	for _, entry := range entries {
		if entry.IsDir() || !isTxFile(entry.Name()) {
			continue
		}

		tx := &Transaction{
			TempPath: filepath.Join(tempDir, entry.Name()),
		}

		if err := tx.load(); err != nil {
			continue // Skip invalid transaction files
		}

		if tx.IsActive() {
			pending = append(pending, tx)
		}
	}

	return pending, nil
}

// isTxFile checks if a filename matches the transaction file pattern
func isTxFile(name string) bool {
	return len(name) > 7 && name[:3] == "tx_" && filepath.Ext(name) == ".json"
}

// BackupFile creates a backup of the original file or directory
func (tx *Transaction) BackupFile(targetPath string) error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(tx.BackupPath), 0700); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	// Use otiai10/copy to handle both files and directories
	err := cp.Copy(targetPath, tx.BackupPath)
	if err != nil {
		return fmt.Errorf("backup file/directory: %w", err)
	}

	return nil
}

// RestoreFromBackup restores the file or directory from backup
func (tx *Transaction) RestoreFromBackup() error {
	// Check if backup file/directory exists
	if _, err := os.Stat(tx.BackupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup does not exist: %w", err)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(tx.File.From), 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	// Use otiai10/copy to restore, overwriting existing content
	err := cp.Copy(tx.BackupPath, tx.File.From)
	if err != nil {
		return fmt.Errorf("restore from backup: %w", err)
	}

	return nil
}
