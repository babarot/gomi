//go:build ignore

// gen.go renders the gomi icon from trash.txt into the favicon files that the
// site and README refer to. Edit trash.txt, then run from the repository root:
//
//	go run docs/icon/gen.go
package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type swatch struct {
	char  byte
	color color.NRGBA
	hex   string
	group string
}

type icon struct {
	palette []swatch
	index   map[byte]int
	grid    []string
	// bounding box of the painted cells
	minX, minY, maxX, maxY int
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ic, err := load(filepath.Join("docs", "icon", "trash.txt"))
	if err != nil {
		return err
	}

	favicon := filepath.Join("docs", "favicon")
	if err := os.WriteFile(filepath.Join(favicon, "favicon.svg"), ic.svg(), 0o644); err != nil {
		return err
	}

	pngs := map[string]int{
		"favicon.png":                  96,
		"favicon-96x96.png":            96,
		"apple-touch-icon.png":         180,
		"web-app-manifest-192x192.png": 192,
		"web-app-manifest-512x512.png": 512,
	}
	for name, size := range pngs {
		b, err := encodePNG(ic.render(size))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(favicon, name), b, 0o644); err != nil {
			return err
		}
	}

	ico, err := ic.ico(16, 32, 48)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(favicon, "favicon.ico"), ico, 0o644); err != nil {
		return err
	}

	// index.html and 404.html refer to these at the site root
	for _, name := range []string{"favicon.ico", "favicon.png"} {
		b, err := os.ReadFile(filepath.Join(favicon, name))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join("docs", name), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func load(path string) (*icon, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ic := &icon{index: map[byte]int{}}
	section := ""
	s := bufio.NewScanner(f)
	for n := 1; s.Scan(); n++ {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == "palette:" || line == "grid:" {
			section = line
			continue
		}
		switch section {
		case "palette:":
			fields := strings.Fields(line)
			if len(fields) != 3 || len(fields[0]) != 1 || fields[0] == "." {
				return nil, fmt.Errorf("%s:%d: want \"<char> <#rrggbb> <group>\"", path, n)
			}
			c, err := parseHex(fields[1])
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %w", path, n, err)
			}
			if _, ok := ic.index[fields[0][0]]; ok {
				return nil, fmt.Errorf("%s:%d: %q is defined twice", path, n, fields[0])
			}
			ic.index[fields[0][0]] = len(ic.palette)
			ic.palette = append(ic.palette, swatch{char: fields[0][0], color: c, hex: fields[1], group: fields[2]})
		case "grid:":
			if len(ic.grid) > 0 && len(line) != len(ic.grid[0]) {
				return nil, fmt.Errorf("%s:%d: row is %d wide, want %d", path, n, len(line), len(ic.grid[0]))
			}
			for i := range len(line) {
				if _, ok := ic.index[line[i]]; !ok && line[i] != '.' {
					return nil, fmt.Errorf("%s:%d: %q is not in the palette", path, n, line[i])
				}
			}
			ic.grid = append(ic.grid, line)
		default:
			return nil, fmt.Errorf("%s:%d: line outside of palette: or grid:", path, n)
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	ic.minX, ic.minY, ic.maxX, ic.maxY = -1, -1, -1, -1
	for y, row := range ic.grid {
		for x := range len(row) {
			if row[x] == '.' {
				continue
			}
			if ic.minX < 0 || x < ic.minX {
				ic.minX = x
			}
			if ic.minY < 0 {
				ic.minY = y
			}
			ic.maxX = max(ic.maxX, x)
			ic.maxY = y
		}
	}
	if ic.minX < 0 {
		return nil, fmt.Errorf("%s: grid has no painted cells", path)
	}
	return ic, nil
}

func parseHex(s string) (color.NRGBA, error) {
	if len(s) != 7 || s[0] != '#' {
		return color.NRGBA{}, fmt.Errorf("bad color %q", s)
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("bad color %q", s)
	}
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}, nil
}

// at returns the swatch of the cell at (x, y), or nil if it is empty.
func (ic *icon) at(x, y int) *swatch {
	if y < 0 || y >= len(ic.grid) || x < 0 || x >= len(ic.grid[y]) || ic.grid[y][x] == '.' {
		return nil
	}
	return &ic.palette[ic.index[ic.grid[y][x]]]
}

// frame is the square, in cells, that the painted area is centered in.
func (ic *icon) frame() (x, y, side float64) {
	w, h := ic.maxX-ic.minX+1, ic.maxY-ic.minY+1
	side = float64(max(w, h))
	return float64(ic.minX) - (side-float64(w))/2, float64(ic.minY) - (side-float64(h))/2, side
}

// svg draws one path per palette entry, grouped by the entry's group, with
// each horizontal run of cells merged into a single rectangle.
func (ic *icon) svg() []byte {
	fx, fy, side := ic.frame()
	var b bytes.Buffer
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%g %g %g %g" shape-rendering="crispEdges">`+"\n", fx, fy, side, side)
	var groups []string
	for _, sw := range ic.palette {
		if !slices.Contains(groups, sw.group) {
			groups = append(groups, sw.group)
		}
	}
	for _, g := range groups {
		fmt.Fprintf(&b, `<g id="%s">`+"\n", g)
		for _, sw := range ic.palette {
			if sw.group != g {
				continue
			}
			var d strings.Builder
			for y, row := range ic.grid {
				for x := 0; x < len(row); {
					if row[x] != sw.char {
						x++
						continue
					}
					start := x
					for x < len(row) && row[x] == sw.char {
						x++
					}
					fmt.Fprintf(&d, "M%d %dh%dv1h-%dz", start, y, x-start, x-start)
				}
			}
			if d.Len() > 0 {
				fmt.Fprintf(&b, `<path fill="%s" d="%s"/>`+"\n", sw.hex, d.String())
			}
		}
		b.WriteString("</g>\n")
	}
	b.WriteString("</svg>\n")
	return b.Bytes()
}

// render draws the icon on a transparent size x size canvas. When a cell fits
// in at least one pixel, cells are scaled by the largest whole number so the
// edges stay sharp; otherwise each pixel averages the cells it covers.
func (ic *icon) render(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	w, h := ic.maxX-ic.minX+1, ic.maxY-ic.minY+1
	if scale := size / max(w, h); scale >= 1 {
		ox, oy := (size-w*scale)/2, (size-h*scale)/2
		for py := range h * scale {
			for px := range w * scale {
				if sw := ic.at(ic.minX+px/scale, ic.minY+py/scale); sw != nil {
					img.SetNRGBA(ox+px, oy+py, sw.color)
				}
			}
		}
		return img
	}

	const ss = 8 // samples per pixel on each axis
	fx, fy, side := ic.frame()
	step := side / float64(size) / ss
	for py := range size {
		for px := range size {
			var r, g, b, a float64
			for sy := range ss {
				for sx := range ss {
					cx := fx + (float64(px*ss+sx)+0.5)*step
					cy := fy + (float64(py*ss+sy)+0.5)*step
					if cx < 0 || cy < 0 {
						continue
					}
					if sw := ic.at(int(cx), int(cy)); sw != nil {
						r += float64(sw.color.R)
						g += float64(sw.color.G)
						b += float64(sw.color.B)
						a++
					}
				}
			}
			if a == 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				R: uint8(r/a + 0.5),
				G: uint8(g/a + 0.5),
				B: uint8(b/a + 0.5),
				A: uint8(a/(ss*ss)*255 + 0.5),
			})
		}
	}
	return img
}

func encodePNG(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ico packs PNG images of the given sizes into an ICO file.
func (ic *icon) ico(sizes ...int) ([]byte, error) {
	var images [][]byte
	for _, size := range sizes {
		b, err := encodePNG(ic.render(size))
		if err != nil {
			return nil, err
		}
		images = append(images, b)
	}

	var b bytes.Buffer
	le := binary.LittleEndian
	_ = binary.Write(&b, le, [3]uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)
	for i, size := range sizes {
		_ = binary.Write(&b, le, struct {
			Width, Height, Colors, Reserved uint8
			Planes, BitCount                uint16
			Size, Offset                    uint32
		}{uint8(size), uint8(size), 0, 0, 1, 32, uint32(len(images[i])), uint32(offset)})
		offset += len(images[i])
	}
	for _, img := range images {
		b.Write(img)
	}
	return b.Bytes(), nil
}
