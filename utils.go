package main

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/fatih/color"
)

func DirSize(path string) (int64, error) {
	var size int64
	var mu sync.Mutex

	// Function to calculate size for a given path
	var calculateSize func(string) error
	calculateSize = func(p string) error {
		fileInfo, err := os.Lstat(p)
		if err != nil {
			return err
		}

		// Skip symbolic links to avoid counting them multiple times
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		if fileInfo.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			for _, entry := range entries {
				if err := calculateSize(filepath.Join(p, entry.Name())); err != nil {
					return err
				}
			}
		} else {
			mu.Lock()
			size += fileInfo.Size()
			mu.Unlock()
		}
		return nil
	}

	// Start calculation from the root path
	if err := calculateSize(path); err != nil {
		return 0, err
	}

	return size, nil
}

// Must panics if the input predicate is false.
func Must(pred bool, msg string, args ...any) {
	if !pred {
		panic(fmt.Sprintf(msg, args...))
	}
}

const logErrKey = "err"

// LogErrAttr wraps an error into a loggable attribute.
func LogErrAttr(err error) slog.Attr {
	if err == nil {
		return slog.Group(logErrKey)
	}
	return slog.String(logErrKey, err.Error())
}

func highlight(content, filename, colorscheme string) (string, error) {
	defer color.Unset()
	var l chroma.Lexer
	l = lexers.Get(filename)
	if l == nil {
		l = lexers.Analyse(content)
	}
	if l == nil {
		l = lexers.Fallback
	}
	style := styles.Get(colorscheme)
	switch {
	case style == nil:
		slog.Warn("theme %s for lighlight not found. fallback to monokai")
		style = styles.Get("monokai")
	case style.Name == "swapoff":
		slog.Warn("theme %s for lighlight not found. defaults to be fallbacked")
		style = styles.Get("monokai")
	}
	var buf bytes.Buffer
	slog.Debug("highlight", "lexer", l.Config().Name, "colorscheme", style.Name)
	if err := quick.Highlight(&buf, content, l.Config().Name, "terminal16m", style.Name); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func runBash(input string) (string, int, error) {
	cmd := exec.Command("bash", "-c", input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if errStr := stderr.String(); errStr != "" {
		output = errStr
		slog.Warn("command might be failed",
			slog.String("command", input),
			slog.String("output", output),
		)
	}
	if err == nil {
		return output, 0, nil
	}
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		return output, -1, err
	}
	return output, ee.ExitCode(), nil
}
