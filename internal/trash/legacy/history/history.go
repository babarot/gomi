package history

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/k0kubun/pp/v3"
	"github.com/rs/xid"
)

const (
	historyVersion = 1
	Filename       = "history.json"
)

// History represents the history of deleted files
type History struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`

	config config.History
	home   string
	path   string
}

type File struct {
	Name      string    `json:"name"`
	ID        string    `json:"id"`
	RunID     string    `json:"group_id"` // to keep backward compatible
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
}

func (f File) GetName() string {
	return f.Name
}

func (f File) GetPath() string {
	return f.To
}

func (f File) GetDeletedAt() time.Time {
	return f.Timestamp
}

func New(home string, c config.History) History {
	if home == "" {
		home = filepath.Join(os.Getenv("HOME"), ".gomi")
	}
	return History{
		home:   home,
		path:   filepath.Join(home, Filename),
		config: c,
	}
}

func (h *History) Open() error {
	slog.Debug("opening history file", "path", h.path)
	defer func() {
		_ = h.backup()
		slog.Debug("backed up")
	}()

	parentDir := filepath.Dir(h.path)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		slog.Warn("mkdir", "dir", parentDir)
		_ = os.Mkdir(parentDir, 0755)
	}

	if _, err := os.Stat(h.path); os.IsNotExist(err) {
		backupFile := h.path + ".backup"
		slog.Warn("history file not found", "path", h.path)
		if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
			slog.Warn("backup file found! attempting to restore from backup", "path", backupFile)
			err := os.Rename(backupFile, h.path)
			if err != nil {
				return fmt.Errorf("failed to restore history from backup: %w", err)
			}
			slog.Debug("successfully restored history from backup")
		}
	}

	f, err := os.OpenFile(h.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("err", "error", err)
		return err
	}
	defer f.Close()

	if stat, err := f.Stat(); err == nil && stat.Size() == 0 {
		slog.Warn("history is empty")
		return nil
	}

	if err := json.NewDecoder(f).Decode(&h); err != nil {
		slog.Error("err", "error", err)
		return err
	}

	return nil
}

func (h *History) backup() error {
	backupFile := h.path + ".backup"
	slog.Debug("backing up history", "path", backupFile)
	f, err := os.Create(backupFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(&h)
}

// Update updates the history by appending the given files to the existing ones.
// It overwrites the current history file with the updated data and sets the version before saving.
// A backup of the current state is created before the update is applied.
func (h *History) Update(files []File) error {
	slog.Debug("updating history file", "path", h.path)
	defer func() {
		_ = h.backup()
		slog.Debug("backed up")
	}()
	f, err := os.Create(h.path)
	if err != nil {
		return err
	}
	defer f.Close()
	h.Files = append(h.Files, files...)
	h.setVersion()
	return json.NewEncoder(f).Encode(&h)
}

// Save saves the current history to the file, overwriting the existing data.
// A backup is performed before saving the history to ensure the current state is preserved.
// Unlike 'update', it does not modify the list of files or set the version.
func (h *History) Save() error {
	slog.Debug("saving history file", "path", h.path)
	defer func() {
		_ = h.backup()
		slog.Debug("backed up")
	}()
	f, err := os.Create(h.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(&h)
}

func (h *History) Remove(target File) error {
	defer func() {
		_ = h.backup()
		slog.Debug("backed up")
	}()
	slog.Debug("deleting file from history file", "path", h.path, "file", target)
	var files []File
	for _, file := range h.Files {
		if file.ID == target.ID {
			slog.Debug("target file found", "id", file.ID, "name", file.Name)
			continue
		}
		files = append(files, file)
	}
	h.Files = files
	return h.Save()
}

func (h *History) setVersion() {
	if h.Version == 0 {
		h.Version = historyVersion
	}
}

func (h History) Filter() []File {
	opts := trash.FilterOptions{
		Include: h.config.Include,
		Exclude: h.config.Exclude,
	}
	slog.Debug("legacy filter items", "len(files)", len(h.Files))
	return trash.Filter(h.Files, opts)
}

func (h History) FileInfo(runID string, arg string) (File, error) {
	name := filepath.Base(arg)
	from, err := filepath.Abs(arg)
	if err != nil {
		return File{}, err
	}
	id := xid.New().String()
	now := time.Now()
	return File{
		Name:  name,
		ID:    id,
		RunID: runID,
		From:  from,
		To: filepath.Join(
			h.home,
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
			runID,
			fmt.Sprintf("%s.%s", name, id),
		),
		Timestamp: now,
	}, nil
}

func (f File) String() string {
	p := pp.New()
	p.SetColoringEnabled(false)
	return p.Sprint(f)
}

// FindByID finds a file in the history by its ID
func (h History) FindByID(id string) *File {
	for _, f := range h.Files {
		if f.ID == id {
			return &f
		}
	}
	return nil
}

// RemoveByPath removes a file from the history by its original path
func (h *History) RemoveByPath(path string) {
	var filtered []File
	for _, f := range h.Files {
		if f.To != path {
			filtered = append(filtered, f)
		}
	}
	h.Files = filtered
}

// Add adds a file to the history
func (h *History) Add(file File) {
	h.Files = append(h.Files, file)
}
