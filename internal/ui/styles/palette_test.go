package styles

import (
	"reflect"
	"testing"

	"github.com/babarot/gomi/internal/config"
)

// TestNewPalette_EveryColor checks that every theme the config accepts
// gives every color of the palette, for both backgrounds, so that nothing
// is drawn without a color by accident.
func TestNewPalette_EveryColor(t *testing.T) {
	for _, name := range append(config.ThemeNames, "") {
		// an empty ui.style, as in a config that leaves the colors out
		p := NewPalette(config.UI{Theme: name})
		v := reflect.ValueOf(p)
		for i := range v.NumField() {
			c, ok := reflect.TypeAssert[Color](v.Field(i))
			if !ok {
				t.Fatalf("%s is not a Color", v.Type().Field(i).Name)
			}
			if c.Light == "" || c.Dark == "" {
				t.Errorf("theme %q: %s has no color", name, v.Type().Field(i).Name)
			}
		}
	}
}

func TestNewPalette_Style(t *testing.T) {
	style := config.StyleConfig{ListView: config.ListViewConfig{Cursor: "#123456"}}

	// ui.style adjusts the gomi theme, with or without its name
	for _, name := range []string{"", "gomi"} {
		if got := NewPalette(config.UI{Theme: name, Style: style}).Cursor; got != same("#123456") {
			t.Errorf("theme %q: Cursor = %v, want the ui.style color", name, got)
		}
	}
	// and no other theme
	if got := NewPalette(config.UI{Theme: "nord", Style: style}).Cursor; got == same("#123456") {
		t.Error("nord: Cursor is the ui.style color")
	}
}
