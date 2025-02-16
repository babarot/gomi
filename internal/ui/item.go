package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/styles"
	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/utils/fs"
	"github.com/babarot/gomi/internal/utils/shell"
	"github.com/dustin/go-humanize"
	"github.com/gabriel-vasile/mimetype"
)

// Item represents a file item in the trash that can be displayed in the list.
type Item struct {
	file            *trash.File
	preview         string
	previewErr      error
	dirListCommand  string
	syntaxHighlight bool
	colorscheme     string
}

// NewItem creates a new Item with the given trash file and configuration.
func NewItem(file *trash.File, cfg config.UI) *Item {
	return &Item{
		file:            file,
		dirListCommand:  cfg.Preview.DirectoryCommand,
		syntaxHighlight: cfg.Preview.SyntaxHighlight,
		colorscheme:     cfg.Preview.Colorscheme,
	}
}

// Title returns the name of the file. If the file is a directory, it appends a slash.
func (i *Item) Title() string {
	fi, err := os.Stat(i.file.TrashPath)
	if err != nil {
		return i.file.Name + "?"
	}
	if fi.IsDir() {
		return i.file.Name + "/"
	}
	return i.file.Name
}

// Description returns the deletion time and original path of the file.
func (i *Item) Description() string {
	_, err := os.Stat(i.file.TrashPath)
	if os.IsNotExist(err) {
		return "(already might have been deleted or unmounted)"
	}

	return fmt.Sprintf("%s • %s",
		humanize.Time(i.file.DeletedAt),
		filepath.Dir(i.file.OriginalPath),
	)
}

// FilterValue returns the string used for filtering the item in the list.
func (i *Item) FilterValue() string {
	return i.file.Name
}

// Size returns the human-readable size of the file.
func (i *Item) Size() string {
	size, err := fs.DirSize(i.file.TrashPath)
	if err != nil {
		return "(cannot be calculated)"
	}
	return humanize.Bytes(uint64(size))
}

// Preview returns the current preview content and error if any.
func (i *Item) Preview() (string, error) {
	return i.preview, i.previewErr
}

// LoadPreview loads the preview content of the file.
// It caches the result so subsequent calls return the cached content.
func (i *Item) LoadPreview() error {
	if i.preview != "" || i.previewErr != nil {
		return i.previewErr
	}

	preview, err := i.generatePreview()
	if err != nil {
		i.previewErr = err
		return err
	}

	i.preview = preview
	return nil
}

// generatePreview creates a preview of the file content.
func (i *Item) generatePreview() (string, error) {
	fi, err := os.Lstat(i.file.TrashPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if fi.IsDir() {
		return i.generateDirPreview()
	}

	return i.generateFilePreview()
}

// generateDirPreview creates a preview of directory contents.
func (i *Item) generateDirPreview() (string, error) {
	if i.dirListCommand == "" {
		// Fallback to built-in directory listing
		var lines []string
		dirs, err := os.ReadDir(i.file.TrashPath)
		if err != nil {
			return "", err
		}

		for _, dir := range dirs {
			info, err := dir.Info()
			if err != nil {
				continue
			}
			name := dir.Name()
			if info.IsDir() {
				name += "/"
			}
			lines = append(lines,
				fmt.Sprintf("%s %7s  %s",
					info.Mode().String(),
					humanize.Bytes(uint64(info.Size())),
					name,
				),
			)
		}
		return strings.Join(lines, "\n"), nil
	}

	// Use custom directory listing command
	cmd := fmt.Sprintf("cd %s && %s", i.file.TrashPath, i.dirListCommand)
	out, _, err := shell.RunCommand(cmd)
	return out, err
}

// generateFilePreview creates a preview of file contents.
func (i *Item) generateFilePreview() (string, error) {
	mtype, err := mimetype.DetectFile(i.file.TrashPath)
	if err != nil {
		return "", err
	}

	// Only preview text files
	if !mtype.Is("text/plain") && (mtype.Parent() == nil || !mtype.Parent().Is("text/plain")) {
		return "", fmt.Errorf("cannot preview %s files", mtype.String())
	}

	content, err := os.ReadFile(i.file.TrashPath)
	if err != nil {
		return "", err
	}

	if !i.syntaxHighlight {
		return string(content), nil
	}

	return i.highlightContent(string(content))
}

// highlightContent applies syntax highlighting to the content.
func (i *Item) highlightContent(content string) (string, error) {
	var lexer chroma.Lexer
	lexer = lexers.Get(i.file.Name)
	if lexer == nil {
		lexer = lexers.Analyse(content)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	style := styles.Get(i.colorscheme)
	if style == nil || style.Name == "swapoff" {
		style = styles.Get("monokai")
	}

	var buf strings.Builder
	err := quick.Highlight(&buf, content, lexer.Config().Name, "terminal16m", style.Name)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// File returns the underlying trash.File.
func (i *Item) File() *trash.File {
	return i.file
}
