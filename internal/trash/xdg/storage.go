package xdg

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/babarot/gomi/internal/fs"
	"github.com/babarot/gomi/internal/trash"
)

// Storage implements the trash.Storage interface for XDG trash specification
type Storage struct {
	// Home trash location (~/.local/share/Trash)
	homeTrash *trashLocation

	// External trash locations ($topdir/.Trash-$uid)
	externalTrashes []*trashLocation

	// Configuration
	config trash.Config
}

// trashLocation represents a single trash directory
type trashLocation struct {
	// Root directory (e.g., ~/.local/share/Trash or /media/disk/.Trash-1000)
	root string

	// Files directory (root/files)
	filesDir string

	// Info directory (root/info)
	infoDir string

	// Whether this is a home trash directory
	isHome bool
}

// NewStorage creates a new XDG-compliant trash storage
func NewStorage(cfg trash.Config) (trash.Storage, error) {
	slog.Info("initialize xdg storage")

	s := &Storage{config: cfg}

	// Initialize home trash
	home, err := s.initHomeTrash()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize home trash: %w", err)
	}
	s.homeTrash = home

	// Skip external trash initialization if forced to use home trash
	if !cfg.ForceHomeTrash {
		if err := s.scanExternalTrashes(); err != nil {
			// Log error but continue - home trash is still usable
			fmt.Fprintf(os.Stderr, "Warning: failed to scan external trashes: %v\n", err)
		}
	}

	return s, nil
}

func (s *Storage) Info() *trash.StorageInfo {
	return &trash.StorageInfo{
		Location:  trash.LocationHome,
		Root:      s.homeTrash.root,
		Available: true,
		Type:      trash.StorageTypeXDG,
	}
}

func (s *Storage) Put(src string) error {
	// Get absolute path
	abs, err := filepath.Abs(src)
	if err != nil {
		return trash.NewStorageError("put", src, err)
	}

	// Select appropriate trash location
	loc, err := s.selectTrashLocation(abs)
	if err != nil {
		return trash.NewStorageError("put", src, err)
	}

	// Generate unique name in trash
	baseName := filepath.Base(abs)
	trashName := baseName
	counter := 1

	for {
		infoPath := filepath.Join(loc.infoDir, trashName+".trashinfo")
		filePath := filepath.Join(loc.filesDir, trashName)

		// Check if name is already taken
		_, errInfo := os.Stat(infoPath)
		_, errFile := os.Stat(filePath)
		if os.IsNotExist(errInfo) && os.IsNotExist(errFile) {
			break
		}

		// Generate new name with counter
		trashName = fmt.Sprintf("%s_%d", baseName, counter)
		counter++
	}

	// Create .trashinfo file first
	info := &TrashInfo{
		Path:         abs,
		DeletionDate: time.Now(),
	}

	infoPath := filepath.Join(loc.infoDir, trashName+".trashinfo")
	if err := info.Save(infoPath); err != nil {
		return trash.NewStorageError("put", src, fmt.Errorf("failed to save trash info: %w", err))
	}

	// Move file to trash
	dstPath := filepath.Join(loc.filesDir, trashName)
	if err := fs.Move(abs, dstPath, s.config.EnableHomeFallback); err != nil {
		// If move fails, clean up the .trashinfo file
		os.Remove(infoPath)
		return trash.NewStorageError("put", src, fmt.Errorf("failed to move file to trash: %w", err))
	}

	return nil
}

func (s *Storage) List() ([]*trash.File, error) {
	var files []*trash.File

	// List files from home trash
	homeFiles, err := s.listLocation(s.homeTrash)
	if err != nil {
		return nil, trash.NewStorageError("list", "", fmt.Errorf("failed to list home trash: %w", err))
	}
	files = append(files, homeFiles...)

	// List files from external trashes
	for _, loc := range s.externalTrashes {
		extFiles, err := s.listLocation(loc)
		if err != nil {
			// Log error but continue with other locations
			fmt.Fprintf(os.Stderr, "Warning: failed to list external trash %s: %v\n", loc.root, err)
			continue
		}
		files = append(files, extFiles...)
	}

	return s.filter(files), nil
}

func (s *Storage) Restore(file *trash.File, dst string) error {
	if dst == "" {
		dst = file.OriginalPath
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return trash.NewStorageError("restore", dst, err)
	}

	// Move file back
	if err := fs.Move(file.TrashPath, dst, s.config.EnableHomeFallback); err != nil {
		return trash.NewStorageError("restore", dst, err)
	}

	// Remove .trashinfo file
	infoPath := filepath.Join(
		file.TrashPath[:len(file.TrashPath)-len("files/"+filepath.Base(file.TrashPath))],
		"info",
		filepath.Base(file.TrashPath)+".trashinfo",
	)
	if err := os.Remove(infoPath); err != nil {
		// Log error but don't fail - file is already restored
		fmt.Fprintf(os.Stderr, "Warning: failed to remove trash info: %v\n", err)
	}

	return nil
}

func (s *Storage) Remove(file *trash.File) error {
	// Remove the actual file
	if err := os.RemoveAll(file.TrashPath); err != nil {
		return trash.NewStorageError("remove", file.TrashPath, err)
	}

	// Remove .trashinfo file
	infoBaseName := filepath.Base(file.TrashPath) + ".trashinfo"
	infoPath := filepath.Join(
		filepath.Dir(filepath.Dir(file.TrashPath)), // Go up two levels (past "files" dir)
		"info",
		infoBaseName,
	)
	if err := os.Remove(infoPath); err != nil {
		// Log error but don't fail - file is already removed
		fmt.Fprintf(os.Stderr, "Warning: failed to remove trash info: %v\n", err)
	}

	return nil
}

func (s *Storage) initHomeTrash() (*trashLocation, error) {
	var root string
	// if s.config.HomeTrashDir != "" {
	// 	root = s.config.HomeTrashDir
	// } else {
	// First try $XDG_DATA_HOME
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		// Fallback to ~/.local/share
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	root = filepath.Join(dataDir, "Trash")
	// }
	slog.Debug("initHomeTrash", "root", root)

	loc := &trashLocation{
		root:     root,
		filesDir: filepath.Join(root, "files"),
		infoDir:  filepath.Join(root, "info"),
		isHome:   true,
	}

	// Create directories if they don't exist
	if err := os.MkdirAll(loc.filesDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create files directory: %w", err)
	}
	if err := os.MkdirAll(loc.infoDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create info directory: %w", err)
	}

	return loc, nil
}

func (s *Storage) scanExternalTrashes() error {
	// Get all mount points
	mounts, err := getMountPoints()
	if err != nil {
		return fmt.Errorf("failed to get mount points: %w", err)
	}

	uid := os.Getuid()
	uidStr := strconv.Itoa(uid)

	for _, mount := range mounts {
		// Check for $topdir/.Trash/$uid
		trashPath := filepath.Join(mount, ".Trash", uidStr)
		if isValidExternalTrash(trashPath) {
			loc := &trashLocation{
				root:     trashPath,
				filesDir: filepath.Join(trashPath, "files"),
				infoDir:  filepath.Join(trashPath, "info"),
				isHome:   false,
			}
			s.externalTrashes = append(s.externalTrashes, loc)
			continue
		}

		// Check for $topdir/.Trash-$uid
		trashPath = filepath.Join(mount, fmt.Sprintf(".Trash-%d", uid))
		if isValidExternalTrash(trashPath) {
			loc := &trashLocation{
				root:     trashPath,
				filesDir: filepath.Join(trashPath, "files"),
				infoDir:  filepath.Join(trashPath, "info"),
				isHome:   false,
			}
			s.externalTrashes = append(s.externalTrashes, loc)
		}
	}

	return nil
}

func (s *Storage) listLocation(loc *trashLocation) ([]*trash.File, error) {
	var files []*trash.File

	// Read files directory
	entries, err := os.ReadDir(loc.filesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read files directory: %w", err)
	}

	for _, entry := range entries {
		// Load corresponding .trashinfo file
		info, err := loadTrashInfo(filepath.Join(loc.infoDir, entry.Name()+".trashinfo"))
		if err != nil {
			// Skip files without valid info
			continue
		}

		// Get file info
		filePath := filepath.Join(loc.filesDir, entry.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			// Skip inaccessible files
			continue
		}

		file := &trash.File{
			Name:         info.OriginalName,
			OriginalPath: info.Path,
			TrashPath:    filePath,
			DeletedAt:    info.DeletionDate,
			Size:         fileInfo.Size(),
			IsDir:        fileInfo.IsDir(),
			FileMode:     fileInfo.Mode(),
		}
		file.SetStorage(s)
		files = append(files, file)
	}

	return files, nil
}

func (s *Storage) selectTrashLocation(path string) (*trashLocation, error) {
	// Check if file is on the same device as home trash
	sameDevice, err := isOnSameDevice(path, s.homeTrash.root)
	if err == nil && sameDevice {
		return s.homeTrash, nil
	}

	// Look for matching external trash
	for _, ext := range s.externalTrashes {
		sameDevice, err := isOnSameDevice(path, ext.root)
		if err == nil && sameDevice {
			return ext, nil
		}
	}

	// If home fallback is enabled, use home trash
	if s.config.EnableHomeFallback {
		return s.homeTrash, nil
	}

	return nil, trash.ErrCrossDevice
}

func (s *Storage) filter(files []*trash.File) []*trash.File {
	opts := trash.FilterOptions{
		Include: s.config.History.Include,
		Exclude: s.config.History.Exclude,
	}
	return trash.Filter(files, opts)
}
