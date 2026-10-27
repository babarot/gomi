package ui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHalfBlocks(t *testing.T) {
	tests := []struct {
		name       string
		pw, ph     int
		w, h       int
		cols, rows int
	}{
		{name: "wide image fits the width", pw: 1100, ph: 700, w: 37, h: 20, cols: 37, rows: 12},
		{name: "tall image fits the height", pw: 390, ph: 3381, w: 37, h: 8, cols: 2, rows: 8},
		{name: "tall image in a pane of a row", pw: 390, ph: 3381, w: 37, h: 1, cols: 1, rows: 1},
		{name: "small image is not stretched", pw: 4, ph: 4, w: 37, h: 20, cols: 4, rows: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := strings.Split(halfBlocks(image.NewRGBA(image.Rect(0, 0, tt.pw, tt.ph)), tt.w, tt.h), "\n")
			if len(lines) != tt.rows {
				t.Errorf("rows = %d, want %d", len(lines), tt.rows)
			}
			if got := ansi.StringWidth(lines[0]); got != tt.cols {
				t.Errorf("cols = %d, want %d", got, tt.cols)
			}
		})
	}
}

func TestHalfBlock(t *testing.T) {
	red := color.NRGBA{R: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}
	none := color.NRGBA{}
	tests := []struct {
		name        string
		top, bottom color.NRGBA
		want        string
	}{
		{name: "both opaque", top: red, bottom: blue, want: "\x1b[38;2;255;0;0m\x1b[48;2;0;0;255m▀"},
		{name: "bottom transparent", top: red, bottom: none, want: "\x1b[49m\x1b[38;2;255;0;0m▀"},
		{name: "top transparent", top: none, bottom: blue, want: "\x1b[49m\x1b[38;2;0;0;255m▄"},
		{name: "both transparent", top: none, bottom: none, want: "\x1b[0m "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := halfBlock(tt.top, tt.bottom); got != tt.want {
				t.Errorf("halfBlock = %q, want %q", got, tt.want)
			}
		})
	}
}
