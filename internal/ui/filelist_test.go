package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/trash"
)

var listNow = time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

func listFile(name string, ago time.Duration, runID string) File {
	return File{File: &trash.File{
		Name:         name,
		OriginalPath: "/home/user/" + name,
		TrashPath:    "/trash/" + name,
		DeletedAt:    listNow.Add(-ago),
		RunID:        runID,
	}}
}

func testList() *fileList {
	l := newFileList()
	l.setFiles([]File{
		listFile("old", 30*24*time.Hour, ""),
		listFile("a", time.Hour, "run"),
		listFile("b", time.Hour+time.Second, "run"),
		listFile("c", 26*time.Hour, ""),
	}, listNow)
	return &l
}

func TestDayLabel(t *testing.T) {
	tests := []struct {
		ago  time.Duration
		want string
	}{
		{time.Hour, "Today"},
		{12 * time.Hour, "Today"}, // 00:00 the same day
		{13 * time.Hour, "Yesterday"},
		{3 * 24 * time.Hour, "Thursday"},
		{8 * 24 * time.Hour, "Sat, Jun 7"},
	}
	for _, tt := range tests {
		if got := dayLabel(listNow.Add(-tt.ago), listNow); got != tt.want {
			t.Errorf("dayLabel(%v ago) = %q, want %q", tt.ago, got, tt.want)
		}
	}
}

func TestFileList_Rows(t *testing.T) {
	rows := testList().rows(listNow)
	var got []string
	for _, r := range rows {
		switch {
		case r.kind == rowDay:
			got = append(got, r.day)
		case r.batch && r.first:
			got = append(got, "┌"+r.file.Name)
		case r.batch && r.last:
			got = append(got, "└"+r.file.Name)
		default:
			got = append(got, r.file.Name)
		}
	}
	want := []string{"Today", "┌a", "└b", "Yesterday", "c", "Fri, May 16", "old"}
	if len(got) != len(want) {
		t.Fatalf("rows = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rows = %q, want %q", got, want)
		}
	}
}

func TestFileList_MoveSkipsHeadings(t *testing.T) {
	l := testList()
	names := func() string {
		f, _ := l.current(listNow)
		return f.Name
	}
	if got := names(); got != "a" {
		t.Fatalf("cursor starts on %q, want a", got)
	}
	l.move(2, listNow) // past the Yesterday heading
	if got := names(); got != "c" {
		t.Errorf("after 2 down, cursor on %q, want c", got)
	}
	l.move(10, listNow) // stops at the last file
	if got := names(); got != "old" {
		t.Errorf("after 10 down, cursor on %q, want old", got)
	}
	l.toStart(listNow)
	if got := names(); got != "a" {
		t.Errorf("after toStart, cursor on %q, want a", got)
	}
}

func TestFileList_Filter(t *testing.T) {
	l := testList()
	l.filter.SetValue("OL") // case does not matter
	shown := l.shown()
	if len(shown) != 1 || shown[0].Name != "old" {
		t.Errorf("shown = %v, want [old]", shown)
	}
	// the place a file was deleted from matches too
	l.filter.SetValue("home/user")
	if got := len(l.shown()); got != 4 {
		t.Errorf("shown %d files for the place, want 4", got)
	}
}

func TestFileList_SetFilesKeepsCursor(t *testing.T) {
	l := testList()
	l.move(2, listNow) // on c
	files := []File{listFile("c", 26*time.Hour, ""), listFile("old", 30*24*time.Hour, "")}
	l.setFiles(files, listNow) // a and b were deleted
	if f, _ := l.current(listNow); f.Name != "c" {
		t.Errorf("cursor on %q after reloading, want c", f.Name)
	}
}

func TestSameRun(t *testing.T) {
	a, b := listFile("a", 0, ""), listFile("b", 0, "")
	if sameRun(a, b) {
		t.Error("files without a run ID are not one run")
	}
}

func TestUIHeight(t *testing.T) {
	tests := []struct {
		spec string
		term int
		want int
	}{
		{"100%", 40, 40},
		{"50%", 40, 20},
		{"30", 40, 30},
		{"30", 20, 20}, // never taller than the terminal
		{"1", 40, minHeight},
		{"", 40, 40},
	}
	for _, tt := range tests {
		if got := uiHeight(tt.spec, tt.term); got != tt.want {
			t.Errorf("uiHeight(%q, %d) = %d, want %d", tt.spec, tt.term, got, tt.want)
		}
	}
}

func TestShortPath(t *testing.T) {
	tests := []struct {
		path string
		w    int
		want string
	}{
		{"~/projects/gomi", 30, "~/projects/gomi"},
		{"/private/tmp/claude/abcdef/scratchpad/pr", 26, "/private/…/scratchpad/pr"},
		// only the last part fits whole, so the one before it is cut
		{"/private/tmp/claude/fee7ca5a-5176-4dc0/scratchpad", 32, "/private/…/fee7ca5a-…/scratchpad"},
	}
	for _, tt := range tests {
		if got := shortPath(tt.path, tt.w); got != tt.want {
			t.Errorf("shortPath(%q, %d) = %q, want %q", tt.path, tt.w, got, tt.want)
		}
	}
}

func TestMatchRunes(t *testing.T) {
	// Ⱥ takes two bytes and its lower case three, so byte offsets of the
	// lowercased name do not fit the name
	if i, n := matchRunes("Ⱥ.txt", "txt"); i != 2 || n != 3 {
		t.Errorf("matchRunes = %d, %d, want 2, 3", i, n)
	}
	if _, n := matchRunes("main.go", "xyz"); n != 0 {
		t.Errorf("matchRunes found %d runes of a query not there", n)
	}
}

func TestView_UnicodeFilter(t *testing.T) {
	m := newTestModel()
	m.list.setFiles([]File{listFile("Ⱥ.txt", time.Hour, "")}, listNow)
	m.list.filter.SetValue("txt")
	_ = m.View() // must not panic
}

func TestView_ConfirmTallerThanUI(t *testing.T) {
	m := newTestModel()
	m.height = minHeight
	a, b := listFile("a", time.Hour, ""), listFile("b", 2*time.Hour, "")
	m.list.setFiles([]File{a, b}, listNow)
	m.state.OpenConfirm(ConfirmStateTypeYES, []File{a, b})
	view := m.View().Content // must not panic
	if !strings.Contains(view, "YES") {
		t.Errorf("the dialog is not shown whole:\n%s", view)
	}
}

func TestQueueSizes(t *testing.T) {
	m := newTestModel()
	var files []File
	for _, name := range []string{"d1", "d2", "d3"} {
		f := listFile(name, time.Hour, "")
		f.IsDir = true
		files = append(files, f)
	}
	known := listFile("f", time.Hour, "")
	known.SizeKnown = true
	files = append(files, known)

	// two start at once; a file whose size is known is not queued
	if got := len(m.queueSizes(files)); got != listSizers {
		t.Errorf("started %d, want %d", got, listSizers)
	}
	if got := len(m.sizeQueue); got != 1 {
		t.Errorf("queued %d, want 1", got)
	}
	// queueing again, as after a delete, does not start more than there is
	// room for
	if got := len(m.queueSizes(files)); got != 0 {
		t.Errorf("queueing again started %d while %d are being sized", got, listSizers)
	}
	// each one done starts the next
	updated, cmd := m.Update(sizeLoadedMsg{path: "/trash/d1", size: 1, queued: true})
	m = asModel(t, updated)
	if cmd == nil || len(m.sizeQueue) != 0 || m.sizing != listSizers {
		t.Errorf("the next directory did not start: queue %d, sizing %d", len(m.sizeQueue), m.sizing)
	}
}
