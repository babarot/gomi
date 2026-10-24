package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// cellStyle is what an SGR sequence can set on a cell. Colors are kept in a
// canonical form ("#rrggbb", or "c<n>" for an indexed color) so that the same
// color written as 31, 38;5;1 or 38:5:1 compares equal.
type cellStyle struct {
	fg, bg    string
	bold      bool
	faint     bool
	italic    bool
	underline bool
	reverse   bool
	strike    bool
}

type cell struct {
	text  string // one grapheme, or "" for the second half of a wide one
	style cellStyle
}

func (c cell) isSpace() bool { return c.text == " " }

// visible reports whether the cell shows anything: a space shows only
// through its background, reverse, underline or strikethrough.
func (c cell) visible() bool {
	if !c.isSpace() {
		return true
	}
	return c.style.bg != "" || c.style.reverse || c.style.underline || c.style.strike
}

// effective drops what a space cannot show, so two renderings that look the
// same normalize the same.
func (c cell) effective() cellStyle {
	if c.isSpace() {
		return c.style.blank()
	}
	return c.style
}

func (s cellStyle) String() string {
	var attrs []string
	if s.fg != "" {
		attrs = append(attrs, "fg="+s.fg)
	}
	if s.bg != "" {
		attrs = append(attrs, "bg="+s.bg)
	}
	for _, a := range []struct {
		on   bool
		name string
	}{
		{s.bold, "bold"},
		{s.faint, "faint"},
		{s.italic, "italic"},
		{s.underline, "underline"},
		{s.reverse, "reverse"},
		{s.strike, "strike"},
	} {
		if a.on {
			attrs = append(attrs, a.name)
		}
	}
	return strings.Join(attrs, " ")
}

// normalize turns a rendered view into text that changes only when the view
// looks different: the characters on screen, then the runs of styled cells
// on each line. It fails the test on any escape sequence other than SGR,
// which it would otherwise drop silently.
func normalize(t *testing.T, view string) string {
	t.Helper()

	var (
		lines [][]cell
		line  []cell
		cur   cellStyle
		state byte
		p     = ansi.NewParser()
	)
	for len(view) > 0 {
		seq, width, n, newState := ansi.DecodeSequence(view, state, p)
		state = newState
		view = view[n:]
		switch {
		case seq == "\n":
			lines = append(lines, line)
			line = nil
		case seq[0] == ansi.ESC:
			cmd := ansi.Cmd(p.Command())
			if !strings.HasPrefix(seq, "\x1b[") || cmd.Final() != 'm' || cmd.Prefix() != 0 || cmd.Intermediate() != 0 {
				t.Fatalf("unexpected escape sequence %q", seq)
			}
			applySGR(t, &cur, p.Params())
		case width > 0:
			line = append(line, cell{text: seq, style: cur})
			for range width - 1 {
				line = append(line, cell{style: cur})
			}
		default:
			t.Fatalf("unexpected control sequence %q", seq)
		}
	}
	lines = append(lines, line)

	// Trailing cells that show nothing are dropped, and so are trailing
	// empty lines.
	for i, l := range lines {
		end := len(l)
		for end > 0 && !l[end-1].visible() {
			end--
		}
		lines[i] = l[:end]
	}
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	b.WriteString("-- screen --\n")
	for _, l := range lines {
		for _, c := range l {
			b.WriteString(c.text)
		}
		b.WriteString("\n")
	}
	b.WriteString("-- styles --\n")
	for i, l := range lines {
		for _, r := range styleRuns(l) {
			fmt.Fprintf(&b, "L%d %d-%d %s\n", i+1, r.start, r.end, r.style)
		}
	}
	return b.String()
}

type styleRun struct {
	start, end int
	style      cellStyle
}

// blank drops what a space cannot show: bold, italic and faint, and the
// foreground unless reverse, underline or strikethrough draws with it.
func (s cellStyle) blank() cellStyle {
	if !s.reverse && !s.underline && !s.strike {
		s.fg = ""
	}
	s.bold, s.italic, s.faint = false, false, false
	return s
}

// styleRuns returns the runs of styled cells on a line. A space shows only
// part of a style, so it joins the run around it when that part matches: a
// space that shows nothing joins any run, and runs neither start nor end
// with one.
func styleRuns(l []cell) []styleRun {
	var (
		runs    []styleRun
		cur     styleRun
		open    bool
		hasText bool // cur.style was taken from a cell other than a space
	)
	flush := func() {
		if open && cur.style != (cellStyle{}) {
			runs = append(runs, cur)
		}
		open = false
	}
	for i, c := range l {
		if !c.visible() {
			continue
		}
		fits := false
		switch {
		case !open:
		case c.isSpace():
			fits = c.effective() == cur.style.blank()
		case hasText:
			fits = c.effective() == cur.style
		default:
			fits = c.effective().blank() == cur.style
		}
		if !fits {
			flush()
			cur, open, hasText = styleRun{start: i, style: c.effective()}, true, !c.isSpace()
		} else if !c.isSpace() && !hasText {
			cur.style, hasText = c.effective(), true
		}
		cur.end = i + 1
	}
	flush()
	return runs
}

// applySGR applies the parameters of one SGR sequence to s.
func applySGR(t *testing.T, s *cellStyle, params ansi.Params) {
	t.Helper()

	ps := make([]int, len(params))
	for i, p := range params {
		ps[i] = p.Param(0)
	}
	if len(ps) == 0 {
		ps = []int{0}
	}
	for i := 0; i < len(ps); i++ {
		switch p := ps[i]; {
		case p == 0:
			*s = cellStyle{}
		case p == 1:
			s.bold = true
		case p == 2:
			s.faint = true
		case p == 3:
			s.italic = true
		case p == 4:
			s.underline = true
		case p == 7:
			s.reverse = true
		case p == 9:
			s.strike = true
		case p == 22:
			s.bold, s.faint = false, false
		case p == 23:
			s.italic = false
		case p == 24:
			s.underline = false
		case p == 27:
			s.reverse = false
		case p == 29:
			s.strike = false
		case 30 <= p && p <= 37:
			s.fg = fmt.Sprintf("c%d", p-30)
		case 90 <= p && p <= 97:
			s.fg = fmt.Sprintf("c%d", p-90+8)
		case 40 <= p && p <= 47:
			s.bg = fmt.Sprintf("c%d", p-40)
		case 100 <= p && p <= 107:
			s.bg = fmt.Sprintf("c%d", p-100+8)
		case p == 39:
			s.fg = ""
		case p == 49:
			s.bg = ""
		case p == 38 || p == 48:
			var c string
			c, i = extendedColor(t, ps, i)
			if p == 38 {
				s.fg = c
			} else {
				s.bg = c
			}
		default:
			t.Fatalf("unexpected SGR parameter %d in %v", p, ps)
		}
	}
}

// extendedColor reads the color of a 38 or 48 parameter at ps[i] and
// returns it with the index of its last parameter.
func extendedColor(t *testing.T, ps []int, i int) (string, int) {
	t.Helper()

	switch {
	case i+2 < len(ps) && ps[i+1] == 5:
		return fmt.Sprintf("c%d", ps[i+2]), i + 2
	case i+4 < len(ps) && ps[i+1] == 2:
		return fmt.Sprintf("#%02x%02x%02x", ps[i+2], ps[i+3], ps[i+4]), i + 4
	}
	t.Fatalf("malformed extended color in %v", ps)
	return "", i
}

func TestSameView(t *testing.T) {
	want := "-- screen --\nfoo\n-- styles --\nL1 0-3 fg=#ad58b4 bg=#1c1c1c bold\n"
	for _, tt := range []struct {
		got  string
		same bool
	}{
		{"-- screen --\nfoo\n-- styles --\nL1 0-3 fg=#ad58b4 bg=#1c1c1c bold\n", true},
		{"-- screen --\nfoo\n-- styles --\nL1 0-3 fg=#ad58b3 bg=#1d1b1c bold\n", true},
		{"-- screen --\nfoo\n-- styles --\nL1 0-3 fg=#ad58b2 bg=#1c1c1c bold\n", false},
		{"-- screen --\nfoo\n-- styles --\nL1 0-3 bg=#ad58b4 bg=#1c1c1c bold\n", false},
		{"-- screen --\nfoo\n-- styles --\nL1 0-3 fg=#ad58b4 bg=#1c1c1c\n", false},
		{"-- screen --\nfoo\n-- styles --\nL1 0-4 fg=#ad58b4 bg=#1c1c1c bold\n", false},
		{"-- screen --\nfoe\n-- styles --\nL1 0-3 fg=#ad58b4 bg=#1c1c1c bold\n", false},
	} {
		if got := sameView(tt.got, want); got != tt.same {
			t.Errorf("sameView(%q) = %v, want %v", tt.got, got, tt.same)
		}
	}
	// Text on the screen that looks like a color is still compared exactly.
	if sameView("-- screen --\nfg=#000001\n-- styles --\n", "-- screen --\nfg=#000000\n-- styles --\n") {
		t.Error("sameView allowed a difference in the screen text")
	}
}
