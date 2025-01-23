package ui

import (
	"bytes"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/quick"
	"github.com/alecthomas/chroma/styles"
	"github.com/fatih/color"
)

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
