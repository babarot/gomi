package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/babarot/gomi/internal/fs"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/utils/shell"

	"al.essio.dev/pkg/shellescape"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/styles"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/gabriel-vasile/mimetype"
)

type File struct {
	*trash.File

	dirListCommand  string
	syntaxHighlight bool
	colorscheme     string
}

func (f File) isSelected() bool {
	return selectionManager.Contains(f)
}

func (f File) Description() string {
	_, err := os.Stat(f.File.TrashPath)
	if os.IsNotExist(err) {
		return "(already might have been deleted)"
	}

	return fmt.Sprintf("%s %s %s",
		humanize.Time(f.File.DeletedAt),
		bullet,
		filepath.Dir(f.File.OriginalPath),
	)
}

func (f File) Title() string {
	fi, err := os.Stat(f.File.TrashPath)
	if err != nil {
		return f.File.Name + "?"
	}
	if fi.IsDir() {
		return f.File.Name + "/"
	}
	return f.File.Name
}

func (f File) FilterValue() string {
	return f.File.Name
}

func (f File) Size() string {
	var sizeStr string
	size, err := fs.DirSize(f.TrashPath)
	if err != nil {
		sizeStr = "(cannot be calculated)"
	} else {
		sizeStr = humanize.Bytes(uint64(size))
	}
	return sizeStr
}

func (f File) Browse() (string, error) {
	var content string

	fi, err := os.Lstat(f.TrashPath)
	if err != nil {
		slog.Debug("no such file", "file", f.TrashPath)
		return content, errCannotPreview
	}
	if fi.IsDir() {
		if f.dirListCommand == "" {
			slog.Debug("preview dir command is not set, fallback to builtin dir func")
			lines := []string{}
			dirs, _ := os.ReadDir(f.TrashPath)
			for _, dir := range dirs {
				info, _ := dir.Info()
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
		input := fmt.Sprintf("cd %s; %s", shellescape.Quote(f.TrashPath), f.dirListCommand)
		slog.Debug("command to list dir", "input", input)
		out, _, err := shell.RunCommand(input)
		if err != nil {
			slog.Error("command failed", "command", input, "error", err)
		}
		return out, err
	}
	mtype, err := mimetype.DetectFile(f.TrashPath)
	if err != nil {
		return content, err
	}
	if mtype.Is("text/plain") || (mtype.Parent() != nil && mtype.Parent().Is("text/plain")) {
		// ok
	} else {
		slog.Debug("cannot preview", "mimetype", mtype.String())
		return content, errCannotPreview
	}
	fp, err := os.Open(f.TrashPath)
	if err != nil {
		return content, errCannotPreview
	}
	defer fp.Close()
	var fileContent strings.Builder
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		fileContent.WriteString(scanner.Text() + "\n")
	}
	if err := scanner.Err(); err != nil {
		return content, err
	}
	content = fileContent.String()
	if f.syntaxHighlight {
		content, _ = f.colorize(content)
	}
	return content, nil
}

func (f File) colorize(content string) (string, error) {
	defer color.Unset()
	var l chroma.Lexer
	l = lexers.Get(f.Name)
	if l == nil {
		l = lexers.Analyse(content)
	}
	if l == nil {
		slog.Debug("highlight: fallback to default lexer")
		l = lexers.Fallback
	}
	style := styles.Get(f.colorscheme)
	switch {
	case style == nil:
		style = styles.Get("monokai")
	case style.Name == "swapoff":
		style = styles.Get("monokai")
	}
	var buf bytes.Buffer
	if err := quick.Highlight(&buf, content, l.Config().Name, "terminal16m", style.Name); err != nil {
		return "", err
	}
	return buf.String(), nil
}
