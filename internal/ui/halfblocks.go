package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"golang.org/x/image/draw"
)

// halfCell is the pixels of a cell drawn with half blocks: the upper one
// in the foreground of ▀ and the lower one in its background
var halfCell = image.Pt(1, 2)

// halfBlocks draws img in at most w columns by h rows of half blocks. The
// image keeps its aspect ratio and is not stretched, and it takes a cell
// however small it is. A pixel less than half opaque is left to the
// terminal's background.
func halfBlocks(img image.Image, w, h int) string {
	b := img.Bounds()
	cols, rows := imageCells(b.Dx(), b.Dy(), max(1, w), max(1, h), halfCell)
	dst := image.NewNRGBA(image.Rect(0, 0, cols*halfCell.X, rows*halfCell.Y))
	// CatmullRom keeps thin lines that ApproxBiLinear drops, but reads
	// every pixel of a large image; it scales only the last 4 times
	if mid := dst.Bounds().Size().Mul(4); b.Dx() > mid.X && b.Dy() > mid.Y {
		m := image.NewNRGBA(image.Rectangle{Max: mid})
		draw.ApproxBiLinear.Scale(m, m.Bounds(), img, b, draw.Src, nil)
		img, b = m, m.Bounds()
	}
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)

	lines := make([]string, rows)
	for r := range lines {
		var s strings.Builder
		for x := range cols {
			s.WriteString(halfBlock(dst.NRGBAAt(x, 2*r), dst.NRGBAAt(x, 2*r+1)))
		}
		s.WriteString("\x1b[0m")
		lines[r] = s.String()
	}
	return strings.Join(lines, "\n")
}

// halfBlock draws the pixels top and bottom in a cell
func halfBlock(top, bottom color.NRGBA) string {
	switch t, b := top.A >= 0x80, bottom.A >= 0x80; {
	case t && b:
		return fgColor(top) + bgColor(bottom) + "▀"
	case t:
		return "\x1b[49m" + fgColor(top) + "▀"
	case b:
		return "\x1b[49m" + fgColor(bottom) + "▄"
	default:
		return "\x1b[0m "
	}
}

func fgColor(c color.NRGBA) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B) }
func bgColor(c color.NRGBA) string { return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B) }
