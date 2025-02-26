package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	imgcolor "image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/utils/fs"
	"github.com/babarot/gomi/internal/utils/shell"

	"al.essio.dev/pkg/shellescape"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/styles"
	"github.com/dustin/go-humanize"
	"github.com/eliukblau/pixterm/pkg/ansimage"
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
		return "(already might have been deleted or unmounted)"
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
		return content, ErrCannotPreview
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
		slog.Debug("run ls-like command", "input", input)
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
	if isImageFile(mtype.String()) {
		slog.Debug("file is an image, trying to preview", "mimetype", mtype.String())
		// Preview the image to match the height of the preview panel
		// defaultHeight(30) - 11 - 1 = 18, adjusting to a height of 18
		return f.previewImage(f.TrashPath, defaultWidth, 18)
	}
	if mtype.Is("text/plain") || (mtype.Parent() != nil && mtype.Parent().Is("text/plain")) {
		// ok
	} else {
		slog.Debug("cannot preview", "mimetype", mtype.String())
		return content, ErrCannotPreview
	}
	fp, err := os.Open(f.TrashPath)
	if err != nil {
		return content, ErrCannotPreview
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

// isImageFile checks whether the given MIME type is an image
func isImageFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// previewImage converts an image file for terminal display
func (f File) previewImage(path string, maxWidth, maxHeight int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Get the original image dimensions.
	bounds := img.Bounds()
	origWidth, origHeight := bounds.Dx(), bounds.Dy()
	slog.Debug("reading image",
		"path", path,
		"width", origWidth,
		"height", origHeight,
	)

	// Set the background color to transparent.
	bgColor := imgcolor.RGBA{0, 0, 0, 0}
	height := max(defaultHeight, maxHeight*2+2)

	pix, err := ansimage.NewScaledFromImage(
		img,
		height,
		maxWidth,
		bgColor,
		ansimage.ScaleModeFit,
		ansimage.NoDithering,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create ANSI image: %w", err)
	}

	// Get the ANSI image as a string.
	ansiStr := pix.Render()
	// Append image information, but keep it concise
	info := fmt.Sprintf("\nImage: %dx%d pixels", origWidth, origHeight)

	return ansiStr + info, nil
}
