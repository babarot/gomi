package trash

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Manager handles multiple trash storage implementations
type Manager struct {
	storages []Storage
	config   *Config
}

// Config holds configuration for the trash manager
type Config struct {
	// Enable XDG trash storage
	EnableXDG bool

	// Enable Legacy trash storage (~/.gomi)
	EnableLegacy bool

	// Enable fallback to home trash when external trash fails
	EnableHomeFallback bool

	// Custom home trash directory (optional)
	HomeTrashDir string

	// Verbose output
	Verbose bool
}

// NewManager creates a new trash manager with the given configuration
func NewManager(cfg *Config) (*Manager, error) {
	var storages []Storage

	// Initialize storages based on configuration
	if cfg.EnableXDG {
		xdgStorage, err := newXDGStorage(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize XDG storage: %w", err)
		}
		storages = append(storages, xdgStorage)
	}

	if cfg.EnableLegacy {
		legacyStorage, err := newLegacyStorage(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize legacy storage: %w", err)
		}
		storages = append(storages, legacyStorage)
	}

	if len(storages) == 0 {
		return nil, errors.New("no storage backend configured")
	}

	return &Manager{
		storages: storages,
		config:   cfg,
	}, nil
}

// Put moves the file at src path to trash
func (m *Manager) Put(src string) error {
	// Get absolute path
	absPath, err := filepath.Abs(src)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	fi, err := os.Lstat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Select appropriate storage
	storage, err := m.selectStorageForPath(absPath)
	if err != nil {
		if !m.config.EnableHomeFallback {
			return err
		}
		// Try fallback to home storage
		storage, err = m.getHomeStorage()
		if err != nil {
			return fmt.Errorf("failed to get home storage for fallback: %w", err)
		}
		if m.config.Verbose {
			fmt.Printf("Falling back to home storage for %s\n", absPath)
		}
	}

	// Try to put the file
	if err := storage.Put(absPath); err != nil {
		return fmt.Errorf("failed to put file: %w", err)
	}

	if m.config.Verbose {
		if fi.IsDir() {
			fmt.Printf("Moved directory %s to trash\n", absPath)
		} else {
			fmt.Printf("Moved file %s to trash\n", absPath)
		}
	}

	return nil
}

// List returns all files from all storages
func (m *Manager) List() ([]*File, error) {
	var allFiles []*File

	for _, storage := range m.storages {
		files, err := storage.List()
		if err != nil {
			// Log error but continue with other storages
			fmt.Fprintf(os.Stderr, "Warning: failed to list files from storage: %v\n", err)
			continue
		}
		allFiles = append(allFiles, files...)
	}

	return allFiles, nil
}

// Restore restores the given file
func (m *Manager) Restore(file *File, dst string) error {
	if file.storage == nil {
		return errors.New("file has no associated storage")
	}
	return file.storage.Restore(file, dst)
}

// Remove permanently removes the file from trash
func (m *Manager) Remove(file *File) error {
	if file.storage == nil {
		return errors.New("file has no associated storage")
	}
	return file.storage.Remove(file)
}

// selectStorageForPath selects the appropriate storage for the given path
func (m *Manager) selectStorageForPath(path string) (Storage, error) {
	// First storage that accepts the path wins
	for _, s := range m.storages {
		info := s.Info()
		if !info.Available {
			continue
		}

		// For XDG storage, check if path is on the same device
		if info.Location == LocationExternal {
			// Implementation will check if the path is on the same device
			if err := m.checkSameDevice(path, info.Root); err == nil {
				return s, nil
			}
			continue
		}

		// For home storage or legacy storage
		if info.Location == LocationHome {
			return s, nil
		}
	}

	return nil, errors.New("no suitable storage found for path")
}

// getHomeStorage returns the home directory storage
func (m *Manager) getHomeStorage() (Storage, error) {
	for _, s := range m.storages {
		info := s.Info()
		if info.Location == LocationHome && info.Available {
			return s, nil
		}
	}
	return nil, errors.New("home storage not available")
}

// checkSameDevice checks if two paths are on the same device
func (m *Manager) checkSameDevice(path1, path2 string) error {
	stat1, err := os.Stat(path1)
	if err != nil {
		return err
	}

	stat2, err := os.Stat(path2)
	if err != nil {
		return err
	}

	// Type assertion to get sys-specific fields
	sys1, ok1 := stat1.Sys().(*syscall.Stat_t)
	sys2, ok2 := stat2.Sys().(*syscall.Stat_t)
	if !ok1 || !ok2 {
		return errors.New("failed to get device information")
	}

	if sys1.Dev != sys2.Dev {
		return errors.New("paths are on different devices")
	}

	return nil
}
