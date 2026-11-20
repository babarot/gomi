package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// mouseModel is a list of files deleted today and yesterday, drawn in 20
// rows from the terminal's row 5, with no preview beside it
func mouseModel(t *testing.T) Model {
	t.Helper()
	m := newTestModel()
	m.config.Preview.Position = "none"
	m.height, m.termHeight, m.cursorY = 20, 40, 5
	var files []File
	for i, name := range []string{"a", "b", "c", "d", "e"} {
		files = append(files, listFile(name, time.Duration(i+1)*time.Hour, name))
	}
	files = append(files, listFile("old", 30*time.Hour, "old"))
	m.list.setFiles(files, listNow)
	return m
}

// rowOf is the terminal's row the line showing name is drawn on
func rowOf(t *testing.T, m Model, name string) int {
	t.Helper()
	for i, line := range strings.Split(ansi.Strip(m.listView()), "\n") {
		if strings.Contains(line, " "+name+" ") {
			return m.frameTop() + i
		}
	}
	t.Fatalf("%s is not drawn:\n%s", name, ansi.Strip(m.listView()))
	return 0
}

func click(x, y int) tea.MouseClickMsg {
	return tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft}
}

func wheel(b tea.MouseButton, x, y int) tea.MouseWheelMsg {
	return tea.MouseWheelMsg{X: x, Y: y, Button: b}
}

func withNow(t *testing.T) {
	t.Helper()
	f := now
	now = func() time.Time { return listNow }
	t.Cleanup(func() { now = f })
}

func TestMouse_ClickPutsCursorOnRow(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	updated, _ := m.Update(click(20, rowOf(t, m, "c")))
	m = asModel(t, updated)
	if f, _ := m.list.current(listNow); f.Name != "c" {
		t.Errorf("cursor on %q, want c", f.Name)
	}
}

func TestMouse_ClickWhenTheTerminalScrolled(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	m.cursorY = 35 // 20 rows do not fit below; the terminal scrolls them up
	if got := m.frameTop(); got != 20 {
		t.Fatalf("frameTop %d, want 20", got)
	}
	updated, _ := m.Update(click(20, rowOf(t, m, "d")))
	if f, _ := asModel(t, updated).list.current(listNow); f.Name != "d" {
		t.Errorf("cursor on %q, want d", f.Name)
	}
}

func TestMouse_ClickOnHeadingOrOutsideDoesNothing(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	before := m.list.cursor
	for _, y := range []int{
		m.frameTop(),     // the header
		m.frameTop() + 1, // Today's heading
		m.frameTop() + m.height - 1,
		0, // above the UI
	} {
		updated, _ := m.Update(click(20, y))
		if got := asModel(t, updated).list.cursor; got != before {
			t.Errorf("a click on row %d moved the cursor to %d", y, got)
		}
	}
}

func TestMouse_NothingWhileSomethingHasTheFocus(t *testing.T) {
	withNow(t)
	for name, set := range map[string]func(*Model){
		"help":     func(m *Model) { m.helpOpen = true },
		"sort":     func(m *Model) { m.sortMenu = true },
		"confirm":  func(m *Model) { m.openDeleteConfirm([]File{listFile("a", time.Hour, "a")}) },
		"no cpr":   func(m *Model) { m.cursorY = -1 },
		"detail":   func(m *Model) { m.state.SetView(DetailView) },
		"quitting": func(m *Model) { m.state.SetView(Quitting) },
	} {
		m := mouseModel(t)
		y := rowOf(t, m, "c")
		before := m.list.cursor
		set(&m)
		for _, msg := range []tea.Msg{click(20, y), wheel(tea.MouseWheelDown, 20, y)} {
			updated, _ := m.Update(msg)
			if got := asModel(t, updated).list.cursor; got != before {
				t.Errorf("%s: %T moved the cursor", name, msg)
			}
		}
	}
}

func TestMouse_WheelMovesCursor(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	y := rowOf(t, m, "b")
	for range 3 {
		updated, _ := m.Update(wheel(tea.MouseWheelDown, 20, y))
		m = asModel(t, updated)
	}
	if f, _ := m.list.current(listNow); f.Name != "d" {
		t.Errorf("after 3 down, cursor on %q, want d", f.Name)
	}
	updated, _ := m.Update(wheel(tea.MouseWheelUp, 20, y))
	if f, _ := asModel(t, updated).list.current(listNow); f.Name != "c" {
		t.Errorf("after 1 up, cursor on %q, want c", f.Name)
	}
}

func TestMouse_WheelOverPaneScrollsIt(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	m.config.Preview.Position = "right"
	m.width = 140
	m, _ = m.followCursor(nil)
	f, _ := m.list.current(listNow)
	content := strings.Repeat("line\n", 100)
	m.previews[f.TrashPath] = previewResult{available: true, content: content}
	m.viewport.SetContent(content)
	m.resizePreview()

	l := m.listLayout()
	y := m.frameTop() + 5
	before := m.list.cursor
	updated, _ := m.Update(wheel(tea.MouseWheelDown, l.listW+5, y))
	m = asModel(t, updated)
	if m.viewport.YOffset() != wheelLines {
		t.Errorf("pane scrolled to %d, want %d", m.viewport.YOffset(), wheelLines)
	}
	if m.list.cursor != before {
		t.Errorf("the wheel over the pane moved the cursor")
	}
}

func TestMouse_WheelScrollsDetail(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	m.state.SetView(DetailView)
	m.viewport.SetContent(strings.Repeat("line\n", 100))
	m.resizePreview()
	updated, _ := m.updateMouse(wheel(tea.MouseWheelDown, 10, m.frameTop()+10))
	m = asModel(t, updated)
	if got := m.viewport.YOffset(); got != wheelLines {
		t.Errorf("detail scrolled to %d, want %d", got, wheelLines)
	}
}

func TestMouse_ModeOnlyWhereTheMouseActs(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Errorf("the list does not take the mouse")
	}
	m.cursorY = -1
	if m.View().MouseMode != tea.MouseModeNone {
		t.Errorf("the mouse is taken where the UI's rows are not known")
	}
}

func TestMouse_ClickOnViewGoesToNextView(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	header := ansi.Strip(m.headerLine())
	end := ansi.StringWidth(header)
	label := ansi.StringWidth(m.viewLabel())
	y := m.frameTop()

	// left of "view: recent" stays
	updated, _ := m.Update(click(end-label-2, y))
	if got := asModel(t, updated).list.view; got != viewRecent {
		t.Errorf("a click beside the view's name went to %s", got)
	}
	for _, x := range []int{end - label, end - 1} {
		m := mouseModel(t)
		want := m.nextView() // the list is shared with what Update returns
		updated, _ := m.Update(click(x, y))
		if got := asModel(t, updated).list.view; got != want {
			t.Errorf("a click at column %d: view %s, want %s", x, got, want)
		}
	}
}

func TestMouse_ClickOnViewToPickerFocusesFilter(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	m.list.setView(viewBatches, listNow)
	end := ansi.StringWidth(ansi.Strip(m.headerLine()))
	updated, _ := m.Update(click(end-1, m.frameTop()))
	m = asModel(t, updated)
	if m.list.view != viewPicker || !m.list.filtering {
		t.Errorf("view %s, filtering %v; want picker with the focus in the filter", m.list.view, m.list.filtering)
	}
}

func TestMouse_PickerTakesTheMouseWhileTyping(t *testing.T) {
	withNow(t)
	m := mouseModel(t)
	m.list.setView(viewPicker, listNow)
	m.list.filtering = true
	m.list.filter.Focus()
	m.list.filter.SetValue("user") // every file was deleted from /home/user

	// the wheel and a click move the cursor; the filter keeps the focus
	// and what was typed
	start, _ := m.list.current(listNow)
	updated, _ := m.Update(wheel(tea.MouseWheelDown, 20, m.frameTop()+3))
	m = asModel(t, updated)
	if f, _ := m.list.current(listNow); f.TrashPath == start.TrashPath {
		t.Errorf("the wheel did not move the cursor in the picker")
	}
	target := m.list.rows(listNow)[0].file
	updated, _ = m.Update(click(20, rowOf(t, m, target.Name)))
	m = asModel(t, updated)
	if f, _ := m.list.current(listNow); f.TrashPath != target.TrashPath {
		t.Errorf("cursor on %q, want %q", f.Name, target.Name)
	}
	if !m.list.filtering || m.list.filter.Value() != "user" {
		t.Errorf("filtering %v, query %q; want the filter to keep the focus and user", m.list.filtering, m.list.filter.Value())
	}

	// a click on the view goes on to the next view, keeping what was typed
	end := ansi.StringWidth(ansi.Strip(m.headerLine()))
	want := m.nextView()
	updated, _ = m.Update(click(end-1, m.frameTop()))
	m = asModel(t, updated)
	if m.list.view != want || m.list.filtering || m.list.filter.Value() != "user" {
		t.Errorf("view %s, filtering %v, query %q; want %s, false, user", m.list.view, m.list.filtering, m.list.filter.Value(), want)
	}
}
