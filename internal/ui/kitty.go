package ui

import (
	"bytes"
	"fmt"
	"image"
	"log/slog"
	"math"
	"os"
	"strings"
	"sync/atomic"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
	"github.com/gabriel-vasile/mimetype"
	"golang.org/x/image/draw"
)

// The ways ui.preview.image draws images: auto uses the kitty graphics
// protocol when the terminal answers kittyQuery, kitty uses it without
// asking, and halfblocks never uses it
const (
	imageAuto       = "auto"
	imageKitty      = "kitty"
	imageHalfBlocks = "halfblocks"
)

// requestCellSize asks the terminal how many pixels a cell is; the answer
// comes as a uv.CellSizeEvent
var requestCellSize = ansi.WindowOp(16)

// kittyQueryID is the image ID the query asks about; the terminal stores
// nothing for a query, so it may be any
const kittyQueryID = 31

// kittyQuery asks the terminal whether it draws images with the kitty
// graphics protocol, with a one pixel image. A terminal that does answers
// with an APC carrying kittyQueryID and OK; one that does not says nothing,
// and neither does a multiplexer that does not pass the protocol through.
var kittyQuery = ansi.KittyGraphics([]byte("AAAA"),
	fmt.Sprintf("i=%d", kittyQueryID), "s=1", "v=1", "a=q", "t=d", "f=24")

// kittyAnswered tells whether e answers kittyQuery with OK
func kittyAnswered(e uv.KittyGraphicsEvent) bool {
	return e.Options.ID == kittyQueryID && bytes.HasPrefix(e.Payload, []byte("OK"))
}

// noPlaceholders tells whether gomi runs in a terminal that answers
// kittyQuery but does not draw the Unicode placeholders: WezTerm with
// enable_kitty_graphics. It would draw the image where the cursor is
// instead. The variables WezTerm sets are looked at, since a multiplexer
// may set TERM_PROGRAM to itself.
func noPlaceholders() bool {
	return os.Getenv("TERM_PROGRAM") == "WezTerm" || os.Getenv("WEZTERM_PANE") != ""
}

// imageIDs numbers the images sent to the terminal. The IDs start from the
// process ID, so that two gomi in the same terminal do not share one.
var imageIDs atomic.Uint32

func init() { imageIDs.Store(uint32(os.Getpid()&0xff) << 16) }

// nextImageID returns a new image ID, which fits in the 24 bits of the
// color the placeholders carry it in
func nextImageID() int {
	for {
		if id := int(imageIDs.Add(1) & 0xffffff); id != 0 {
			return id
		}
	}
}

// kittyPreview reads the image at path for a pane of w columns by h rows of
// cell pixels each; a zero cell is one the terminal did not tell. The
// result holds the sequence that sends the image to the terminal and, as
// its content, the placeholders the terminal draws the image on. It
// returns false when path is not an image that can be decoded.
func kittyPreview(path string, w, h int, cell image.Point) (previewResult, bool) {
	mtype, err := mimetype.DetectFile(path)
	if err != nil || !isImageFile(mtype.String()) {
		return previewResult{}, false
	}
	fp, err := os.Open(path)
	if err != nil {
		return previewResult{}, false
	}
	defer fp.Close()
	img, _, err := image.Decode(fp)
	if err != nil {
		slog.Debug("cannot decode image", "path", path, "error", err)
		return previewResult{}, false
	}
	if w <= 0 || h <= 0 {
		w, h = defaultWidth, defaultHeight
	}
	b := img.Bounds()
	known := cell.X > 0 && cell.Y > 0
	if !known {
		cell = defaultCell
	}
	// the last row tells the size
	cols, rows := imageCells(b.Dx(), b.Dy(), w, max(1, h-1), cell)

	id := nextImageID()
	// an image is not sent with more pixels than the cells show; a cell of
	// unknown size may be on a high density display
	px := image.Pt(cols*cell.X, rows*cell.Y)
	if !known {
		px = px.Mul(2)
	}
	small := shrink(img, px.X, px.Y)
	var seq strings.Builder
	err = kitty.EncodeGraphics(&seq, small, &kitty.Options{
		Action: kitty.TransmitAndPut,
		ID:     id,
		Quiet:  2,
		// EncodeGraphics sends nothing unless told how
		Transmission:     kitty.Direct,
		Format:           kitty.PNG,
		Chunk:            true,
		VirtualPlacement: true,
		// a terminal that draws the image at the cursor still leaves it
		DoNotMoveCursor: true,
		Columns:         cols,
		Rows:            rows,
	})
	if err != nil {
		slog.Debug("cannot encode image", "path", path, "error", err)
		return previewResult{}, false
	}
	content := placeholders(id, cols, rows) + fmt.Sprintf("\nImage: %dx%d pixels", b.Dx(), b.Dy())
	return previewResult{content: content, available: true, graphics: seq.String(), imageID: id}, true
}

// defaultCell is the pixels of a cell the terminal did not tell, twice as
// high as wide like most fonts
var defaultCell = image.Pt(10, 20)

// imageCells returns how many columns and rows an image of pw by ph pixels
// takes to fit in w columns by h rows of cell pixels each, keeping its
// aspect ratio. An image smaller than that is not stretched.
func imageCells(pw, ph, w, h int, cell image.Point) (cols, rows int) {
	h = min(h, len(placeholderDiacritics))
	// the size of the image in cells, scaled down to fit
	iw := float64(pw) / float64(cell.X)
	ih := float64(ph) / float64(cell.Y)
	scale := min(1, float64(w)/iw, float64(h)/ih)
	cols = min(w, max(1, int(math.Round(iw*scale))))
	rows = min(h, max(1, int(math.Round(ih*scale))))
	return cols, rows
}

// shrink scales img down to fit in w by h pixels; a smaller image is
// returned as it is
func shrink(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	if b.Dx() <= w && b.Dy() <= h {
		return img
	}
	// fit while keeping the aspect ratio
	if b.Dx()*h > b.Dy()*w {
		h = max(1, b.Dy()*w/b.Dx())
	} else {
		w = max(1, b.Dx()*h/b.Dy())
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// placeholders returns rows lines of cols placeholders on which the
// terminal draws the image id. The color carries the ID, and the
// diacritics on the first cell of a line its row; the cells after it
// follow on from it.
func placeholders(id, cols, rows int) string {
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", id>>16&0xff, id>>8&0xff, id&0xff)
	rest := strings.Repeat(string(kitty.Placeholder), cols-1)
	lines := make([]string, rows)
	for r := range lines {
		lines[r] = fg + string(kitty.Placeholder) +
			string(placeholderDiacritics[r]) + string(placeholderDiacritics[0]) +
			rest + "\x1b[39m"
	}
	return strings.Join(lines, "\n")
}

// deleteImages returns the sequence that frees the images ids in the
// terminal
func deleteImages(ids []int) string {
	var s strings.Builder
	for _, id := range ids {
		s.WriteString(ansi.KittyGraphics(nil, "a=d", "d=I", fmt.Sprintf("i=%d", id), "q=2"))
	}
	return s.String()
}

// placeholderDiacritics are the combining marks that number the rows and
// columns of the placeholders, from kitty's gen/rowcolumn-diacritics.txt
var placeholderDiacritics = []rune{
	0x0305, 0x030D, 0x030E, 0x0310, 0x0312, 0x033D, 0x033E, 0x033F, 0x0346, 0x034A,
	0x034B, 0x034C, 0x0350, 0x0351, 0x0352, 0x0357, 0x035B, 0x0363, 0x0364, 0x0365,
	0x0366, 0x0367, 0x0368, 0x0369, 0x036A, 0x036B, 0x036C, 0x036D, 0x036E, 0x036F,
	0x0483, 0x0484, 0x0485, 0x0486, 0x0487, 0x0592, 0x0593, 0x0594, 0x0595, 0x0597,
	0x0598, 0x0599, 0x059C, 0x059D, 0x059E, 0x059F, 0x05A0, 0x05A1, 0x05A8, 0x05A9,
	0x05AB, 0x05AC, 0x05AF, 0x05C4, 0x0610, 0x0611, 0x0612, 0x0613, 0x0614, 0x0615,
	0x0616, 0x0617, 0x0657, 0x0658, 0x0659, 0x065A, 0x065B, 0x065D, 0x065E, 0x06D6,
	0x06D7, 0x06D8, 0x06D9, 0x06DA, 0x06DB, 0x06DC, 0x06DF, 0x06E0, 0x06E1, 0x06E2,
	0x06E4, 0x06E7, 0x06E8, 0x06EB, 0x06EC, 0x0730, 0x0732, 0x0733, 0x0735, 0x0736,
	0x073A, 0x073D, 0x073F, 0x0740, 0x0741, 0x0743, 0x0745, 0x0747, 0x0749, 0x074A,
	0x07EB, 0x07EC, 0x07ED, 0x07EE, 0x07EF, 0x07F0, 0x07F1, 0x07F3, 0x0816, 0x0817,
	0x0818, 0x0819, 0x081B, 0x081C, 0x081D, 0x081E, 0x081F, 0x0820, 0x0821, 0x0822,
	0x0823, 0x0825, 0x0826, 0x0827, 0x0829, 0x082A, 0x082B, 0x082C, 0x082D, 0x0951,
	0x0953, 0x0954, 0x0F82, 0x0F83, 0x0F86, 0x0F87, 0x135D, 0x135E, 0x135F, 0x17DD,
	0x193A, 0x1A17, 0x1A75, 0x1A76, 0x1A77, 0x1A78, 0x1A79, 0x1A7A, 0x1A7B, 0x1A7C,
	0x1B6B, 0x1B6D, 0x1B6E, 0x1B6F, 0x1B70, 0x1B71, 0x1B72, 0x1B73, 0x1CD0, 0x1CD1,
	0x1CD2, 0x1CDA, 0x1CDB, 0x1CE0, 0x1DC0, 0x1DC1, 0x1DC3, 0x1DC4, 0x1DC5, 0x1DC6,
	0x1DC7, 0x1DC8, 0x1DC9, 0x1DCB, 0x1DCC, 0x1DD1, 0x1DD2, 0x1DD3, 0x1DD4, 0x1DD5,
	0x1DD6, 0x1DD7, 0x1DD8, 0x1DD9, 0x1DDA, 0x1DDB, 0x1DDC, 0x1DDD, 0x1DDE, 0x1DDF,
	0x1DE0, 0x1DE1, 0x1DE2, 0x1DE3, 0x1DE4, 0x1DE5, 0x1DE6, 0x1DFE, 0x20D0, 0x20D1,
	0x20D4, 0x20D5, 0x20D6, 0x20D7, 0x20DB, 0x20DC, 0x20E1, 0x20E7, 0x20E9, 0x20F0,
	0x2CEF, 0x2CF0, 0x2CF1, 0x2DE0, 0x2DE1, 0x2DE2, 0x2DE3, 0x2DE4, 0x2DE5, 0x2DE6,
	0x2DE7, 0x2DE8, 0x2DE9, 0x2DEA, 0x2DEB, 0x2DEC, 0x2DED, 0x2DEE, 0x2DEF, 0x2DF0,
	0x2DF1, 0x2DF2, 0x2DF3, 0x2DF4, 0x2DF5, 0x2DF6, 0x2DF7, 0x2DF8, 0x2DF9, 0x2DFA,
	0x2DFB, 0x2DFC, 0x2DFD, 0x2DFE, 0x2DFF, 0xA66F, 0xA67C, 0xA67D, 0xA6F0, 0xA6F1,
	0xA8E0, 0xA8E1, 0xA8E2, 0xA8E3, 0xA8E4, 0xA8E5, 0xA8E6, 0xA8E7, 0xA8E8, 0xA8E9,
	0xA8EA, 0xA8EB, 0xA8EC, 0xA8ED, 0xA8EE, 0xA8EF, 0xA8F0, 0xA8F1, 0xAAB0, 0xAAB2,
	0xAAB3, 0xAAB7, 0xAAB8, 0xAABE, 0xAABF, 0xAAC1, 0xFE20, 0xFE21, 0xFE22, 0xFE23,
	0xFE24, 0xFE25, 0xFE26, 0x10A0F, 0x10A38, 0x1D185, 0x1D186, 0x1D187, 0x1D188, 0x1D189,
	0x1D1AA, 0x1D1AB, 0x1D1AC, 0x1D1AD, 0x1D242, 0x1D243, 0x1D244,
}
