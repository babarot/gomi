package xdg

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/utils/fs"
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

	// Mount point root path for resolving relative paths
	mountRoot string
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
	trashes := []string{s.homeTrash.root}
	for _, ext := range s.externalTrashes {
		trashes = append(trashes, ext.root)
	}
	return &trash.StorageInfo{
		Location:  trash.LocationHome,
		Trashes:   trashes,
		Available: true,
		Type:      trash.StorageTypeXDG,
	}
}

func (s *Storage) Put(src string) error {
	abs, err := filepath.Abs(src)
	if err != nil {
		return trash.NewStorageError("put", src, err)
	}

	// Select appropriate trash location
	loc, err := s.selectTrashLocation(abs)
	if err != nil {
		return trash.NewStorageError("put", src, err)
	}

	info := &TrashInfo{
		Path:         abs,
		MountRoot:    loc.mountRoot,
		DeletionDate: time.Now(),
		RunID:        s.config.RunID,
	}

	// Reserve a unique name in the trash by creating its .trashinfo file.
	// Save creates it with O_EXCL, so of two puts that pick the same name at
	// once (gomi puts its arguments in parallel), one wins and the other
	// moves on to the next name. Checking first with Stat would leave a gap
	// between the check and the creation for the other to slip into.
	baseName := filepath.Base(abs)
	trashName := baseName
	var infoPath string
	for counter := 1; ; counter++ {
		infoPath = filepath.Join(loc.infoDir, trashName+".trashinfo")
		// A file without its .trashinfo (left by another tool) also holds the name
		if _, err := os.Lstat(filepath.Join(loc.filesDir, trashName)); err == nil {
			trashName = fmt.Sprintf("%s_%d", baseName, counter)
			continue
		} else if !os.IsNotExist(err) {
			return trash.NewStorageError("put", src, err)
		}
		err := info.Save(infoPath)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return trash.NewStorageError("put", src, fmt.Errorf("failed to save trash info: %w", err))
		}
		trashName = fmt.Sprintf("%s_%d", baseName, counter)
	}

	// Move file to trash
	dstPath := filepath.Join(loc.filesDir, trashName)
	if err := fs.Move(abs, dstPath, s.config.HomeFallback); err != nil {
		// The file is in the trash in full, so its .trashinfo is kept for
		// it to be listed; only what is left at src is reported
		if !errors.Is(err, fs.ErrSourceRemains) {
			os.Remove(infoPath)
		}
		return trash.NewStorageError("put", src, fmt.Errorf("failed to move file to trash: %w", err))
	}

	return nil
}

func (s *Storage) List() ([]*trash.File, error) {
	var files []*trash.File

	start := time.Now()

	// With a size filter, each trash's directorysizes tells the disk usage
	// of the directories measured before; the filter measures the others
	// and the caches are written back once it is done
	var caches []*sizeCache
	list := func(loc *trashLocation) ([]*trash.File, error) {
		var c *sizeCache
		if size := s.config.History.Exclude.Size; size.Min != "" || size.Max != "" {
			c = newSizeCache(loc.root)
		}
		files, err := s.listLocation(loc, c)
		// a trash that could not be listed keeps its cache as it is
		if err == nil && c != nil {
			caches = append(caches, c)
		}
		return files, err
	}

	// List files from home trash
	homeFiles, err := list(s.homeTrash)
	if err != nil {
		return nil, trash.NewStorageError("list", "", fmt.Errorf("failed to list home trash: %w", err))
	}
	files = append(files, homeFiles...)

	// List files from external trashes
	for _, loc := range s.externalTrashes {
		slog.Debug("listing external trash",
			"path", loc.root)
		extFiles, err := list(loc)
		if err != nil {
			// Log warning but continue with other locations
			slog.Warn("failed to list external trash",
				"path", loc.root,
				"error", err)
			continue
		}
		slog.Debug("external trash contents",
			"path", loc.root,
			"files", len(extFiles))
		files = append(files, extFiles...)
	}

	// the caches look at what was given to the filter, so the directories
	// it dropped for their size are saved too
	filtered := s.filter(files)
	for _, c := range caches {
		c.save()
	}
	slog.Debug("xdg storage listed",
		"files", len(files),
		"took", time.Since(start))
	return filtered, nil
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
	if err := fs.Move(file.TrashPath, dst, s.config.HomeFallback); err != nil {
		return trash.NewStorageError("restore", dst, err)
	}

	// Remove .trashinfo file
	if err := os.Remove(infoPathForFile(file.TrashPath)); err != nil {
		slog.Warn("failed to remove trash info", "error", err)
	}

	return nil
}

func (s *Storage) Remove(file *trash.File) error {
	// Remove the actual file
	if err := os.RemoveAll(file.TrashPath); err != nil {
		return trash.NewStorageError("remove", file.TrashPath, err)
	}

	// Remove .trashinfo file
	if err := os.Remove(infoPathForFile(file.TrashPath)); err != nil {
		slog.Warn("failed to remove trash info", "error", err)
	}

	return nil
}

// infoPathForFile returns the .trashinfo file path for a given trash file path.
// Given a path like /trash/files/foo.txt, it returns /trash/info/foo.txt.trashinfo.
func infoPathForFile(trashPath string) string {
	return filepath.Join(
		filepath.Dir(filepath.Dir(trashPath)), // Go up two levels (past "files" dir)
		"info",
		filepath.Base(trashPath)+".trashinfo",
	)
}

func (s *Storage) initHomeTrash() (*trashLocation, error) {
	var root string

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

	loc := &trashLocation{
		root:     root,
		filesDir: filepath.Join(root, "files"),
		infoDir:  filepath.Join(root, "info"),
		isHome:   true,
		// Home trash doesn't need mount root as it always uses absolute paths
		mountRoot: "",
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
				root:      trashPath,
				filesDir:  filepath.Join(trashPath, "files"),
				infoDir:   filepath.Join(trashPath, "info"),
				isHome:    false,
				mountRoot: mount, // Set mount root for relative paths
			}
			s.externalTrashes = append(s.externalTrashes, loc)
			continue
		}

		// Check for $topdir/.Trash-$uid
		trashPath = filepath.Join(mount, fmt.Sprintf(".Trash-%d", uid))
		if isValidExternalTrash(trashPath) {
			loc := &trashLocation{
				root:      trashPath,
				filesDir:  filepath.Join(trashPath, "files"),
				infoDir:   filepath.Join(trashPath, "info"),
				isHome:    false,
				mountRoot: mount, // Set mount root for relative paths
			}
			s.externalTrashes = append(s.externalTrashes, loc)
		}
	}

	return nil
}

// infoReaders is how many .trashinfo files are read at once. Opening one
// can wait on a security scanner that inspects every open, which takes
// most of the time of a listing when its caches are cold; the waits of
// files read at once overlap.
const infoReaders = 16

// entryInfo is what reading an entry of files/ found: its .trashinfo and
// the entry itself, or nil when it is skipped
type entryInfo struct {
	info     *TrashInfo
	fileInfo os.FileInfo
}

// readEntry reads the .trashinfo of the entry name in loc, and lstats the
// entry itself, so that a trashed symlink is listed whether or not its
// target exists
func readEntry(loc *trashLocation, name string) entryInfo {
	info, err := loadTrashInfo(filepath.Join(loc.infoDir, name+".trashinfo"))
	if err != nil {
		// Skip files without valid info
		return entryInfo{}
	}
	fileInfo, err := os.Lstat(filepath.Join(loc.filesDir, name))
	if err != nil {
		// Skip inaccessible files
		return entryInfo{}
	}
	return entryInfo{info: info, fileInfo: fileInfo}
}

// readEntries reads the entries of files/ in loc, infoReaders at a time.
// Each result is at the index of its entry, in the order of entries.
func readEntries(loc *trashLocation, entries []os.DirEntry) []entryInfo {
	found := make([]entryInfo, len(entries))
	next := make(chan int)
	var wg sync.WaitGroup
	for range min(infoReaders, len(entries)) {
		wg.Go(func() {
			for i := range next {
				found[i] = readEntry(loc, entries[i].Name())
			}
		})
	}
	for i := range entries {
		next <- i
	}
	close(next)
	wg.Wait()
	return found
}

// listLocation lists the trash at loc. With cache, the directories get the
// size the size filter judges them by from it.
func (s *Storage) listLocation(loc *trashLocation, cache *sizeCache) ([]*trash.File, error) {
	var files []*trash.File

	// Read files directory
	entries, err := os.ReadDir(loc.filesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read files directory: %w", err)
	}

	// the entries are read at once, and the cache is filled one by one
	// after them, as it is not safe to share
	found := readEntries(loc, entries)
	for i, entry := range entries {
		if cache != nil {
			cache.names[entry.Name()] = true
		}

		info, fileInfo := found[i].info, found[i].fileInfo
		if info == nil {
			continue
		}

		// Set mount root for resolving relative paths
		info.setMountRoot(loc.mountRoot)

		// Get absolute path
		origPath := info.GetAbsolutePath()

		filePath := filepath.Join(loc.filesDir, entry.Name())
		infoPath := filepath.Join(loc.infoDir, entry.Name()+".trashinfo")
		file := &trash.File{
			Name:         filepath.Base(origPath),
			OriginalPath: origPath,
			TrashPath:    filePath,
			DeletedAt:    info.DeletionDate,
			RunID:        info.RunID,
			Size:         fileInfo.Size(),
			SizeKnown:    !fileInfo.IsDir(),
			IsDir:        fileInfo.IsDir(),
			FileMode:     fileInfo.Mode(),
		}
		if cache != nil && file.IsDir {
			cache.track(entry.Name(), file, infoPath)
		}
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
	if s.config.HomeFallback {
		return s.homeTrash, nil
	}

	return nil, trash.ErrCrossDevice
}

func (s *Storage) filter(files []*trash.File) []*trash.File {
	opts := trash.FilterOptions{
		Include: s.config.History.Include,
		Exclude: s.config.History.Exclude,
	}
	slog.Debug("xdg filter items", "len(files)", len(files))
	return trash.Filter(files, opts)
}

func FindAllTrashDirectories() ([]string, error) {
	var trashDirs []string

	s := &Storage{}
	homeTrash, err := s.initHomeTrash()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize home trash: %w", err)
	}

	if homeTrash != nil {
		trashDirs = append(trashDirs, homeTrash.root)
	}

	if err := s.scanExternalTrashes(); err != nil {
		return nil, fmt.Errorf("failed to scan external trashes: %w", err)
	}

	for _, ext := range s.externalTrashes {
		trashDirs = append(trashDirs, ext.root)
	}

	return trashDirs, nil
}
