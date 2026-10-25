package input

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The whole placeholder shows while nothing is typed, whatever its
// characters are
func TestPlaceholder(t *testing.T) {
	for _, tt := range []struct{ name, want string }{
		{"notes.md", "notes.md"},
		{"café.md", "café.md"}, // NFD é, drawn composed
		{"メモ.txt", "メモ.txt"},
		{"が.txt", "が.txt"},
	} {
		m := New()
		m.Prompt = "New name:"
		m.Placeholder = tt.name
		m.setup()
		if got, want := ansi.Strip(m.input.View()), "New name: "+tt.want; got != want {
			t.Errorf("placeholder %q drawn as %q, want %q", tt.name, got, want)
		}
	}
}
