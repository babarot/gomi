// Package state keeps what gomi remembers between launches that is not a
// setting: the sizes the preview was dragged to. Losing it loses nothing
// but that, so a file that cannot be read is taken as never written.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Store is the state file.
type Store struct {
	path string
}

// Open is the store in dir.
func Open(dir string) *Store {
	return &Store{path: filepath.Join(dir, "state.json")}
}

// DefaultDir is $XDG_STATE_HOME/gomi, or ~/.local/state/gomi.
func DefaultDir() (string, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, "gomi"), nil
}

// settings are what is remembered.
type settings struct {
	// PaneWidth and PaneHeight are what the preview right of the list and
	// under it were dragged to, or 0 for the size they start at.
	PaneWidth  int `json:"pane_width,omitempty"`
	PaneHeight int `json:"pane_height,omitempty"`
}

func (s *Store) read() settings {
	var st settings
	if b, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(b, &st)
	}
	return st
}

// PaneWidth is the width the preview right of the list was dragged to, or 0.
func (s *Store) PaneWidth() int { return max(0, s.read().PaneWidth) }

// PaneHeight is the height the preview under the list was dragged to, or 0.
func (s *Store) PaneHeight() int { return max(0, s.read().PaneHeight) }

// SavePaneWidth remembers the width of the preview right of the list; 0
// forgets it.
func (s *Store) SavePaneWidth(w int) error {
	st := s.read()
	st.PaneWidth = w
	return s.write(st)
}

// SavePaneHeight remembers the height of the preview under the list; 0
// forgets it.
func (s *Store) SavePaneHeight(h int) error {
	st := s.read()
	st.PaneHeight = h
	return s.write(st)
}

// write replaces the file whole, so that a crash leaves the old one or the
// new one and never a part of it.
func (s *Store) write(st settings) error {
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".state-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), s.path)
}
