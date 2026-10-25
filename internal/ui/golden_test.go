package ui

import (
	"flag"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/cursor"
	tea "charm.land/bubbletea/v2"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
)

var updateGolden = flag.Bool("update", false, "rewrite testdata/golden")

// goldenNow is the fixed time the golden views are rendered at.
var goldenNow = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

// fixture is a file in the trash for the golden views. Files are listed
// newest first, in the order of this slice.
type fixture struct {
	name    string
	content string            // file content; ignored for a directory
	entries map[string]string // directory entries; nil for a file
	ago     time.Duration     // how long before goldenNow it was deleted
	runID   string            // the run that deleted it, shared by files deleted together
	dir     string            // where it was deleted from; empty is ~/projects/gomi
}

var fixtures = []fixture{
	{name: "main.go", ago: 2 * time.Hour, content: `package main

import "fmt"

func main() {
	fmt.Println("hello, gomi")
}
`},
	{name: "notes", ago: 5 * time.Hour, dir: "/home/user/docs", entries: map[string]string{
		"a.txt":   "first note\n",
		"b.md":    "# second note\n\nsome text\n",
		"c.log":   "2025-06-15 started\n2025-06-15 stopped\n",
		"todo.md": "- [ ] restore files\n",
	}},
	{name: "archive.bin", ago: 26 * time.Hour, dir: "/home/user/Downloads", content: "\x00\x01\x02\x03\xff\xfe\xfd\xfc\x00\x00\x10\x20"},
	{name: "long.txt", ago: 50 * time.Hour, content: longText()},
	{name: "README.md", ago: 74 * time.Hour, runID: "run-1", content: "# gomi\n\nReplacement for rm with rich trash.\n"},
	{name: "config.yaml", ago: 74*time.Hour + 30*time.Second, runID: "run-1", content: "core:\n  trash:\n    strategy: auto\n"},
	{name: "a-very-long-file-name-for-testing.txt", ago: 122 * time.Hour, content: "a\n"},
	{name: "another-quite-long-file-name.txt", ago: 146 * time.Hour, content: "b\n"},
	{name: "yet-another-long-file-name-here.txt", ago: 170 * time.Hour, content: "c\n"},
	{name: "todo.txt", ago: 240 * time.Hour, content: "buy milk\n"},
	{name: "report.csv", ago: 480 * time.Hour, dir: "/tmp", content: "name,size\ngomi,1\n"},
	{name: "old.log", ago: 960 * time.Hour, dir: "/tmp", content: "old\n"},
}

// longText has more lines than the preview shows, each shorter than the
// preview is wide.
func longText() string {
	var b strings.Builder
	for i := 1; i <= 40; i++ {
		b.WriteString("line ")
		b.WriteString(string(rune('0' + i/10)))
		b.WriteString(string(rune('0' + i%10)))
		b.WriteString(": lorem ipsum dolor sit amet\n")
	}
	return b.String()
}

// makeFixtures creates the fixtures under trash/ in the current directory
// and returns them as trash files. Paths are relative and modes fixed, so
// that nothing about the machine running the test shows in the views.
func makeFixtures(t *testing.T) []*trash.File {
	t.Helper()

	var files []*trash.File
	for _, f := range fixtures {
		path := filepath.Join("trash", f.name)
		isDir := f.entries != nil
		if isDir {
			if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, content := range f.entries {
				writeFile(t, filepath.Join(path, name), content)
			}
			if err := os.Chmod(path, 0o755); err != nil {
				t.Fatal(err)
			}
		} else {
			writeFile(t, path, f.content)
		}
		dir := f.dir
		if dir == "" {
			dir = "/home/user/projects/gomi"
		}
		files = append(files, &trash.File{
			Name:         f.name,
			OriginalPath: dir + "/" + f.name,
			TrashPath:    path,
			DeletedAt:    goldenNow.Add(-f.ago),
			RunID:        f.runID,
			IsDir:        isDir,
		})
	}
	return files
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
}

// keyMsg returns the key message for a key name as key.Binding spells it
// ("tab", "shift+tab", "space", "D").
func keyMsg(name string) tea.KeyPressMsg {
	switch name {
	case "space", " ":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	}
	r := []rune(name)[0]
	return tea.KeyPressMsg{Code: r, Text: name}
}

// driver feeds a model messages the way tea.Program does, running the
// commands they return and feeding their messages back in.
type driver struct {
	t *testing.T
	m tea.Model
	// loading drops what is read in the background, as if it had not
	// come yet
	loading bool
}

func (d *driver) send(msg tea.Msg) {
	m, cmd := d.m.Update(msg)
	d.m = m
	d.run(cmd)
	// tea.Program draws after each update, and the list learns from the
	// drawing how many rows a page has
	_ = d.m.View()
}

func (d *driver) keys(names ...string) {
	for _, n := range names {
		d.send(keyMsg(n))
	}
}

// run runs cmd and what follows from it. A command that does not return
// soon, such as the cursor's blink timer, is dropped, as are blinks: the
// views are rendered with the cursor in its first state.
func (d *driver) run(cmd tea.Cmd) {
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 1000 {
			d.t.Fatal("commands did not settle")
		}
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		msg, ok := runCmd(c)
		if !ok {
			continue
		}
		switch msg := msg.(type) {
		case nil, tea.QuitMsg, cursor.BlinkMsg:
		case previewLoadedMsg, sizeLoadedMsg:
			if !d.loading {
				m, next := d.m.Update(msg)
				d.m = m
				queue = append(queue, next)
			}
		case tea.BatchMsg:
			queue = append(queue, msg...)
		default:
			m, next := d.m.Update(msg)
			d.m = m
			queue = append(queue, next)
		}
	}
}

// runCmd runs c and reports false if it is a timer. The wait is shorter
// than the cursor's blink (530ms) and far longer than any command that
// returns at once.
func runCmd(c tea.Cmd) (tea.Msg, bool) {
	ch := make(chan tea.Msg, 1)
	go func() { ch <- c() }()
	select {
	case msg := <-ch:
		return msg, true
	case <-time.After(300 * time.Millisecond):
		return nil, false
	}
}

// goldenCase is one view to render: a model, set up, then driven by keys.
type goldenCase struct {
	name string
	// model returns the model to drive; nil means the main UI with the
	// default config, changed by opts.
	model func(t *testing.T, files []*trash.File) tea.Model
	opts  func(o *RenderOptions)
	keys  []string
	// width is the terminal's width; zero is 80
	width int
	// cwd is where gomi was started; empty is a directory with nothing
	// deleted in it
	cwd string
	// loading renders the view before what is read in the background comes
	loading bool
}

func mainModel(files []*trash.File, opts func(o *RenderOptions)) tea.Model {
	cfg := config.NewDefaultConfig()
	o := RenderOptions{Config: cfg.UI}
	// ls output depends on the machine; use the builtin listing.
	o.Config.Preview.DirectoryCommand = ""
	if opts != nil {
		opts(&o)
	}
	return NewModel(nil, files, o)
}

func withDelete(o *RenderOptions) { o.DeleteEnabled = true }

// withStyle sets every color of ui.style: to a value of its own when custom,
// or to none, as for colors left out of a config file.
func withStyle(custom bool) func(o *RenderOptions) {
	return func(o *RenderOptions) {
		o.DeleteEnabled = true
		c := func(v string) string {
			if custom {
				return v
			}
			return ""
		}
		st := &o.Config.Style
		st.DeletionDialog = c("#C01010")
		st.ListView.Cursor = c("#C01020")
		st.ListView.Selected = c("#C01030")
		st.ListView.FilterMatch = c("#C01040")
		st.ListView.FilterPrompt = c("#C01050")
		st.DetailView.Border = c("#C01060")
		st.DetailView.InfoPane.DeletedFrom.Foreground = c("#C01070")
		st.DetailView.InfoPane.DeletedFrom.Background = c("#C01080")
		st.DetailView.InfoPane.DeletedAt.Foreground = c("#C01090")
		st.DetailView.InfoPane.DeletedAt.Background = c("#C010A0")
		st.DetailView.PreviewPane.Border = c("#C010B0")
		st.DetailView.PreviewPane.Size.Foreground = c("#C010C0")
		st.DetailView.PreviewPane.Size.Background = c("#C010D0")
		st.DetailView.PreviewPane.Scroll.Foreground = c("#C010E0")
		st.DetailView.PreviewPane.Scroll.Background = c("#C010F0")
	}
}

// withView starts in the named view
func withView(name string) func(o *RenderOptions) {
	return func(o *RenderOptions) { o.Config.View = name }
}

// withTheme draws with the named theme
func withTheme(name string) func(o *RenderOptions) {
	return func(o *RenderOptions) {
		o.DeleteEnabled = true
		o.Config.Theme = name
	}
}

var goldenCases = []goldenCase{
	// list view
	{name: "list"},
	{name: "list-wide", width: 120},
	{name: "list-wide-down", width: 120, keys: []string{"down"}},
	{name: "list-preview-below", opts: func(o *RenderOptions) { o.Config.Preview.Position = "below" }},
	{name: "list-preview-right", width: 100, opts: func(o *RenderOptions) { o.Config.Preview.Position = "right" }, keys: []string{"down", "down", "down"}},
	{name: "view-places", opts: withView("places")},
	{name: "view-places-close", opts: withView("places"), keys: []string{"h", "j", "l"}},
	{name: "view-batches", opts: withView("batches"), keys: []string{"j", "j", "j", "l", "j"}},
	{name: "view-switch", keys: []string{"j", "v"}},
	{name: "view-fold-select", opts: withView("places"), keys: []string{"tab"}},
	{name: "view-fold-preview", width: 120, opts: withView("batches"), keys: []string{"j", "j", "j"}},
	{name: "view-fold-delete", opts: func(o *RenderOptions) { withView("places")(o); o.DeleteEnabled = true }, keys: []string{"D"}},
	{name: "picker", opts: withView("picker")},
	{name: "picker-typing", opts: withView("picker"), keys: []string{"n", "t", "s"}},
	{name: "picker-path", opts: withView("picker"), keys: []string{"t", "m", "p"}},
	{name: "picker-esc-view", opts: withView("picker"), keys: []string{"r", "e", "esc", "v"}},
	{name: "picker-sort-menu", opts: withView("picker"), keys: []string{"esc", "s", "j", "j"}},
	{name: "picker-sorted", opts: withView("picker"), keys: []string{"esc", "s", "3"}},
	{name: "view-to-picker", keys: []string{"v", "v", "v"}},
	{name: "scope-here", cwd: "/home/user/projects/gomi"},
	{name: "scope-all", cwd: "/home/user/projects/gomi", keys: []string{"."}},
	{name: "scope-places", cwd: "/home/user/docs", opts: withView("places")},
	{name: "scope-nothing-here", cwd: "/home/user/projects/gomi", keys: []string{"/", "z", "z", "z"}},
	{name: "help-overlay", keys: []string{"?"}},
	{name: "help-overlay-detail", width: 120, keys: []string{"space", "?"}},
	{name: "list-preview-none", width: 120, opts: func(o *RenderOptions) { o.Config.Preview.Position = "none" }},
	{name: "list-height", opts: func(o *RenderOptions) { o.Config.Height = "12" }},
	{name: "list-next-page", opts: func(o *RenderOptions) { o.Config.Height = "12" }, keys: []string{"right"}},
	{name: "list-end", keys: []string{"G"}},
	{name: "list-selected", keys: []string{"tab", "tab"}},
	{name: "list-selected-cursor", keys: []string{"tab", "tab", "shift+tab"}},
	{name: "list-filter-empty", keys: []string{"/"}},
	{name: "list-filter-typing", keys: []string{"/", "r", "e"}},
	{name: "list-filter-applied", keys: []string{"/", "r", "e", "enter"}},
	{name: "list-help", keys: []string{"?"}},

	// detail view
	{name: "detail-text", keys: []string{"space"}},
	{name: "detail-dir", keys: []string{"down", "space"}},
	{name: "detail-binary", keys: []string{"down", "down", "space"}},
	{name: "detail-scroll", keys: []string{"down", "down", "down", "space", "d"}},
	{name: "detail-selected", keys: []string{"tab", "up", "space"}},
	{name: "detail-help", keys: []string{"space", "?"}},
	{name: "detail-loading", keys: []string{"down", "space"}, loading: true},
	{name: "detail-taken", keys: []string{"down", "down", "down", "down", "down", "space"}},
	{name: "view-places-marks", width: 120, opts: withView("places"), keys: []string{"G"}},

	// delete confirmation
	{name: "confirm-list", opts: withDelete, keys: []string{"D"}},
	{name: "confirm-detail", opts: withDelete, keys: []string{"space", "D"}},
	{name: "confirm-detail-selected", opts: withDelete, keys: []string{"tab", "tab", "space", "D"}},
	{name: "confirm-yes-empty", opts: withDelete, keys: []string{"tab", "tab", "tab", "D"}},
	{name: "confirm-yes-partial", opts: withDelete, keys: []string{"tab", "tab", "tab", "D", "Y", "E"}},
	{name: "confirm-yes-complete", opts: withDelete, keys: []string{"tab", "tab", "tab", "D", "Y", "E", "S"}},
	{name: "confirm-yes-many", opts: withDelete, keys: []string{
		"down", "down", "down", "down", "down", "down", "tab", "tab", "tab", "D",
	}},

	// every color of ui.style set
	{name: "style-custom-list", opts: withStyle(true), keys: []string{"tab", "tab", "shift+tab"}},
	{name: "style-custom-filter", opts: withStyle(true), keys: []string{"/", "r", "e"}},
	{name: "style-custom-detail", opts: withStyle(true), keys: []string{"tab", "up", "space"}},
	{name: "style-custom-detail-dir", opts: withStyle(true), keys: []string{"down", "space", "d"}},
	{name: "style-custom-confirm", opts: withStyle(true), keys: []string{"tab", "tab", "tab", "D", "Y"}},

	// every color of ui.style left out
	{name: "style-empty-list", opts: withStyle(false), keys: []string{"tab", "tab", "shift+tab"}},
	{name: "style-empty-filter", opts: withStyle(false), keys: []string{"/", "r", "e"}},
	{name: "style-empty-detail", opts: withStyle(false), keys: []string{"tab", "up", "space"}},
	{name: "style-empty-detail-dir", opts: withStyle(false), keys: []string{"down", "space", "d"}},
	{name: "style-empty-confirm", opts: withStyle(false), keys: []string{"tab", "tab", "tab", "D", "Y"}},

	// themes other than gomi
	{name: "theme-nord-list", opts: withTheme("nord"), keys: []string{"tab", "tab", "shift+tab"}},
	{name: "theme-catppuccin-mocha-detail", opts: withTheme("catppuccin-mocha"), keys: []string{"tab", "up", "space"}},
	{name: "theme-catppuccin-latte-confirm", opts: withTheme("catppuccin-latte"), keys: []string{"tab", "tab", "tab", "D", "Y"}},

	// prompts outside the TUI
	{name: "prompt-confirm", model: func(t *testing.T, _ []*trash.File) tea.Model {
		m := newConfirmModel("Restore main.go?")
		return &m
	}},
	{name: "prompt-confirm-yes", model: func(t *testing.T, _ []*trash.File) tea.Model {
		m := newConfirmYesModel("Type YES to delete 3 files:")
		return &m
	}},
	{name: "prompt-confirm-yes-typed", keys: []string{"Y", "E", "S"}, model: func(t *testing.T, _ []*trash.File) tea.Model {
		m := newConfirmYesModel("Type YES to delete 3 files:")
		return &m
	}},
	{name: "prompt-input", model: func(t *testing.T, files []*trash.File) tea.Model {
		m := newInputFilenameModel(files[0])
		return &m
	}},
	{name: "prompt-input-error", keys: []string{"a", " ", "b"}, model: func(t *testing.T, files []*trash.File) tea.Model {
		m := newInputFilenameModel(files[0])
		return &m
	}},
}

func TestGolden(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("paths and file modes render differently on Windows")
	}

	dir, err := filepath.Abs(filepath.Join("testdata", "golden"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	// places under the home directory are written with ~
	t.Setenv("HOME", "/home/user")
	defer func(d time.Duration) { cursorWait = d }(cursorWait)
	cursorWait = 0
	defer func(d time.Duration) { previewDelay = d }(previewDelay)
	previewDelay = 0
	// what restoring meets: config.yaml has a file in its place, and the
	// folder of the files from /tmp is gone
	defer func(f func(string) origin) { checkOrigin = f }(checkOrigin)
	checkOrigin = func(path string) origin {
		switch {
		case strings.HasSuffix(path, "/config.yaml"):
			return originTaken
		case strings.HasPrefix(path, "/tmp/"):
			return originGone
		}
		return originBack
	}
	files := makeFixtures(t)

	defer func(f func() time.Time) { now = f }(now)
	now = func() time.Time { return goldenNow }

	backgrounds := []struct {
		name  string
		color color.Color
	}{
		{"dark", color.Black},
		{"light", color.White},
	}
	for _, bg := range backgrounds {
		for _, c := range goldenCases {
			t.Run(c.name+"."+bg.name, func(t *testing.T) {
				defer func(f func() (string, error)) { getwd = f }(getwd)
				if c.cwd != "" {
					getwd = func() (string, error) { return c.cwd, nil }
				}
				var m tea.Model
				if c.model != nil {
					m = c.model(t, files)
				} else {
					m = mainModel(files, c.opts)
				}
				d := &driver{t: t, m: m, loading: c.loading}
				d.run(d.m.Init())
				d.send(tea.BackgroundColorMsg{Color: bg.color})
				width := c.width
				if width == 0 {
					width = 80
				}
				d.send(tea.WindowSizeMsg{Width: width, Height: 30})
				// the prompt left the cursor at the top
				d.send(tea.CursorPositionMsg{})
				d.keys(c.keys...)
				compareGolden(t, filepath.Join(dir, c.name+"."+bg.name+".golden"), normalize(t, d.m.View().Content))
			})
		}
	}
}

func compareGolden(t *testing.T, path, got string) {
	t.Helper()
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run go test ./internal/ui -run TestGolden -update)", err)
	}
	if !sameView(got, string(want)) {
		t.Errorf("view differs from %s (run go test ./internal/ui -run TestGolden -update to accept)\n--- got ---\n%s--- want ---\n%s", path, got, want)
	}
}

// sameView compares two normalized views, letting each channel of a 24-bit
// color differ by one. Renderers round colors differently (lipgloss v1
// truncates #AD58B4 to #ad58b3), and a difference that small does not show.
func sameView(got, want string) bool {
	if got == want {
		return true
	}
	gl, wl := strings.Split(got, "\n"), strings.Split(want, "\n")
	if len(gl) != len(wl) {
		return false
	}
	styles := false // past "-- styles --", where colors may differ
	for i := range gl {
		if gl[i] == wl[i] {
			styles = styles || gl[i] == "-- styles --"
			continue
		}
		if !styles {
			return false
		}
		gt, wt := strings.Split(gl[i], " "), strings.Split(wl[i], " ")
		if len(gt) != len(wt) {
			return false
		}
		for j := range gt {
			if gt[j] != wt[j] && !closeColors(gt[j], wt[j]) {
				return false
			}
		}
	}
	return true
}

// closeColors reports whether a and b are the same fg= or bg= attribute
// with 24-bit colors no more than one apart in each channel.
func closeColors(a, b string) bool {
	if len(a) != len("fg=#rrggbb") || len(b) != len(a) || a[:4] != b[:4] {
		return false
	}
	if a[:4] != "fg=#" && a[:4] != "bg=#" {
		return false
	}
	for k := 4; k < len(a); k += 2 {
		x, err1 := strconv.ParseUint(a[k:k+2], 16, 8)
		y, err2 := strconv.ParseUint(b[k:k+2], 16, 8)
		if err1 != nil || err2 != nil || max(x, y)-min(x, y) > 1 {
			return false
		}
	}
	return true
}
