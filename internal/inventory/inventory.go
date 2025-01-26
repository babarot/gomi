package inventory

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/utils"
	"github.com/docker/go-units"
	"github.com/gobwas/glob"
	"github.com/k0kubun/pp/v3"
	"github.com/k1LoW/duration"
	"github.com/rs/xid"
	"github.com/samber/lo"
)

const (
	inventoryVersion = 1

	inventoryFile = "inventory.json"
)

var (
	gomiPath      = filepath.Join(os.Getenv("HOME"), ".gomi")
	inventoryPath = filepath.Join(gomiPath, inventoryFile)
)

// Inventory represents the log data of deleted objects
type Inventory struct {
	Version int    `json:"version"`
	Files   []File `json:"files"`

	config config.Inventory
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

func New(c config.Inventory) Inventory {
	return Inventory{path: inventoryPath, config: c}
}

func (i *Inventory) Open() error {
	slog.Debug("open inventory", "path", i.path)
	f, err := os.Open(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(&i); err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("inventory version: %d", i.Version))
	return nil
}

func (i *Inventory) update(files []File) error {
	slog.Debug("update inventory", "path", i.path)
	f, err := os.Create(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = files
	i.setVersion()
	return json.NewEncoder(f).Encode(&i)
}

func (i *Inventory) Save(files []File) error {
	slog.Debug("save inventory", "path", i.path)
	f, err := os.Create(i.path)
	if err != nil {
		return err
	}
	defer f.Close()
	i.Files = append(i.Files, files...)
	i.setVersion()
	return json.NewEncoder(f).Encode(&i)
}

func (i Inventory) Filter() []File {
	// do not overwrite original slices
	// because remove them from inventory file actually
	// when updating inventory
	files := i.Files
	files = lo.Reject(files, func(file File, index int) bool {
		return slices.Contains(i.config.Exclude.Files, file.Name)
	})
	files = lo.Reject(files, func(file File, index int) bool {
		for _, pat := range i.config.Exclude.Patterns {
			if regexp.MustCompile(pat).MatchString(file.Name) {
				return true
			}
		}
		for _, g := range i.config.Exclude.Globs {
			if glob.MustCompile(g).Match(file.Name) {
				return true
			}
		}
		return false
	})
	files = lo.Reject(files, func(file File, index int) bool {
		size, err := utils.DirSize(file.To)
		if err != nil {
			return false // false positive
		}
		if s := i.config.Exclude.Size.Min; s != "" {
			min, err := units.FromHumanSize(s)
			if err != nil {
				return false
			}
			if size <= min {
				return true
			}
		}
		if s := i.config.Exclude.Size.Max; s != "" {
			max, err := units.FromHumanSize(s)
			if err != nil {
				return false
			}
			if max <= size {
				return true
			}
		}
		return false
	})
	files = lo.Filter(files, func(file File, index int) bool {
		if period := i.config.Include.Period; period > 0 {
			d, err := duration.Parse(fmt.Sprintf("%d days", period))
			if err != nil {
				slog.Error("parsing duration failed", "error", err)
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

func (i *Inventory) Remove(target File) error {
	slog.Debug("delete file from inventory", "path", i.path, "file", target)
	var files []File
	for _, file := range i.Files {
		if file.ID == target.ID {
			continue
		}
		files = append(files, file)
	}
	return i.update(files)
}

func (i *Inventory) setVersion() {
	if i.Version == 0 {
		i.Version = inventoryVersion
	}
}

func FileInfo(runID string, arg string) (File, error) {
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
			gomiPath,
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
