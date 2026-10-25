package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/babarot/gomi/internal/ui/styles"
)

// asModel extracts Model from tea.Model with a fatal on type mismatch
func asModel(t *testing.T, m tea.Model) Model {
	t.Helper()
	model, ok := m.(Model)
	if !ok {
		t.Fatalf("unexpected model type: %T", m)
	}
	return model
}

// newTestModel creates a minimal Model for testing Update logic
func newTestModel() Model {
	cfg := config.NewDefaultConfig()
	km := keys.NewKeyMap(keys.KeyMapConfig{DeleteEnabled: true})

	l := newFileList()

	return Model{
		state:     NewViewState(),
		keyMap:    km,
		selection: &SelectionManager{items: []File{}},
		config:    cfg.UI,
		list:      &l,
		width:     80,
		height:    40,
		settled:   true,
		viewport:  viewport.Model{},
		previews:  map[string]previewResult{},
		sizes:     map[string]int64{},
		reading:   map[string]bool{},
		origins:   map[string]origin{},
		styles:    styles.New(styles.NewPalette(cfg.UI), true),
		help:      help.New(),
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.width != 120 {
		t.Errorf("width = %d, want 120", model.width)
	}
}

func TestUpdate_WindowSizeMsg_ShrinksListHeight(t *testing.T) {
	m := newTestModel()

	const termHeight = 20
	msg := tea.WindowSizeMsg{Width: 80, Height: termHeight}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.height > termHeight {
		t.Errorf("height = %d, want <= %d (terminal height); "+
			"the UI is not following the terminal height and will push items off-screen",
			model.height, termHeight)
	}
}

func TestUpdate_WindowSizeMsg_SetsHelpWidth(t *testing.T) {
	m := newTestModel()

	const termWidth = 70
	msg := tea.WindowSizeMsg{Width: termWidth, Height: 20}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.help.Width() == 0 {
		t.Fatal("help.Width = 0; the help model was not informed of the " +
			"terminal width, so its View() will not truncate and may overflow")
	}
	if model.help.Width() > termWidth {
		t.Errorf("help.Width = %d, want <= %d (terminal width)",
			model.help.Width(), termWidth)
	}
}

func TestView_FitsSmallTerminal(t *testing.T) {
	m := newTestModel()
	m.help.Styles = helpStyles(styles.NewPalette(m.config), true)
	m.state.SetView(ListView)

	const (
		termWidth  = 70
		termHeight = 20
	)
	msg := tea.WindowSizeMsg{Width: termWidth, Height: termHeight}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	view := model.View().Content

	if h := lipgloss.Height(view); h > termHeight {
		t.Errorf("View height = %d lines, want <= %d; "+
			"output is taller than the terminal so the top rows scroll off",
			h, termHeight)
	}
	if w := lipgloss.Width(view); w > termWidth {
		t.Errorf("View width = %d cols, want <= %d; "+
			"output is wider than the terminal so the right edge is clipped/wrapped",
			w, termWidth)
	}
}

// TestUpdate_WindowSizeMsg_HeightZero verifies that a synthetic
// WindowSizeMsg with only Width set (sent by confirm-view cancel
// paths in update.go) does not collapse the list height.
func TestUpdate_WindowSizeMsg_HeightZero(t *testing.T) {
	m := newTestModel()

	// First, set a real terminal size.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = asModel(t, updated)
	prevHeight := m.height

	// Then send a width-only message (Height == 0).
	updated, _ = m.Update(tea.WindowSizeMsg{Width: m.width})
	model := asModel(t, updated)

	if model.height != prevHeight {
		t.Errorf("height = %d, want %d; "+
			"width-only WindowSizeMsg must not change the height",
			model.height, prevHeight)
	}
}

func TestUpdate_ErrorMsg(t *testing.T) {
	m := newTestModel()

	msg := errorMsg{err: errors.New("test error")}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if model.err == nil {
		t.Error("err should be set")
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_FileListLoadedMsg(t *testing.T) {
	m := newTestModel()

	file := newTestFile("loaded.txt")
	msg := FileListLoadedMsg{files: []File{file}}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	items := model.list.files
	if len(items) != 1 {
		t.Errorf("items = %d, want 1", len(items))
	}
}

func TestUpdate_FileListLoadedMsg_Error(t *testing.T) {
	m := newTestModel()

	msg := FileListLoadedMsg{err: errors.New("load failed")}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if model.err == nil {
		t.Error("err should be set")
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ListView_Quit(t *testing.T) {
	m := newTestModel()
	m.state.SetView(ListView)

	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ListView_CtrlC(t *testing.T) {
	m := newTestModel()
	m.state.SetView(ListView)

	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_DetailView_EscReturnsToList(t *testing.T) {
	m := newTestModel()
	m.state.SetView(DetailView)

	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != ListView {
		t.Errorf("state = %v, want ListView", model.state.current)
	}
}

func TestUpdate_DetailView_Quit(t *testing.T) {
	m := newTestModel()
	m.state.SetView(DetailView)

	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ConfirmView_YesNo_No(t *testing.T) {
	m := newTestModel()
	// Reset global selection manager for test isolation
	m.selection = &SelectionManager{items: []File{}}

	m.state.OpenConfirm(ConfirmStateYesNo, []File{})

	msg := tea.KeyPressMsg{Code: 'n', Text: "n"}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.state.Confirming() {
		t.Error("should have closed the confirmation after pressing 'n'")
	}
}

func TestUpdate_ConfirmView_YesNo_Quit(t *testing.T) {
	m := newTestModel()
	m.selection = &SelectionManager{items: []File{}}

	m.state.OpenConfirm(ConfirmStateYesNo, []File{})

	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ConfirmView_TypeYES_Backspace(t *testing.T) {
	m := newTestModel()
	m.selection = &SelectionManager{items: []File{}}

	m.state.OpenConfirm(ConfirmStateTypeYES, []File{})

	// Type "Y"
	msg := tea.KeyPressMsg{Code: 'Y', Text: "Y"}
	updated, _ := m.Update(msg)
	m = asModel(t, updated)

	if m.state.confirmation.yesInput != "Y" {
		t.Errorf("yesInput = %q, want %q", m.state.confirmation.yesInput, "Y")
	}

	// Backspace
	msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
	updated, _ = m.Update(msg)
	m = asModel(t, updated)

	if m.state.confirmation.yesInput != "" {
		t.Errorf("yesInput = %q, want empty after backspace", m.state.confirmation.yesInput)
	}
}

func TestUpdate_ConfirmView_TypeYES_Esc(t *testing.T) {
	m := newTestModel()
	m.selection = &SelectionManager{items: []File{}}

	m.state.OpenConfirm(ConfirmStateTypeYES, []File{})

	msg := tea.KeyPressMsg{Code: tea.KeyEscape}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.state.Confirming() {
		t.Error("esc should close the confirmation")
	}
}

func TestUpdate_ListView_Enter_NoFiles(t *testing.T) {
	m := newTestModel()
	m.selection = &SelectionManager{items: []File{}}
	m.state.SetView(ListView)

	// Add an item to the list so SelectedItem works
	f := newTestFile("test.txt")
	m.list.setFiles([]File{f}, now())

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	// With no selection manager items, it should pick the current item
	if len(model.choices) != 1 {
		t.Errorf("choices = %d, want 1", len(model.choices))
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ListView_Enter_WithSelection(t *testing.T) {
	m := newTestModel()

	f1 := newTestFile("a.txt")
	f2 := newTestFile("b.txt")
	m.selection = &SelectionManager{items: []File{f1, f2}}
	m.state.SetView(ListView)
	m.list.setFiles([]File{f1, f2}, now())

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := asModel(t, updated)

	if len(model.choices) != 2 {
		t.Errorf("choices = %d, want 2", len(model.choices))
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ShowDetailMsg(t *testing.T) {
	m := newTestModel()

	file := File{File: &trash.File{Name: "detail.txt", TrashPath: "/trash/detail.txt"}}
	msg := ShowDetailMsg{file: file}
	updated, _ := m.Update(msg)
	model := asModel(t, updated)

	if model.state.current != DetailView {
		t.Errorf("state = %v, want DetailView", model.state.current)
	}
	if model.detailFile.Name != "detail.txt" {
		t.Errorf("detailFile.Name = %q, want %q", model.detailFile.Name, "detail.txt")
	}
}

func TestUpdate_ShowDetailMsg_ReadsOnce(t *testing.T) {
	m := newTestModel()
	dir := File{File: &trash.File{Name: "d", TrashPath: "/trash/d", IsDir: true}}
	file := File{File: &trash.File{Name: "f", TrashPath: "/trash/f", Size: 5, SizeKnown: true}}

	// a directory's preview and size are read; a file's size is known
	if got := len(m.loadDetail(dir)); got != 2 {
		t.Errorf("commands for a directory = %d, want 2", got)
	}
	if got := len(m.loadDetail(file)); got != 1 {
		t.Errorf("commands for a file = %d, want 1", got)
	}

	// a read that has started is not started again
	if got := len(m.loadDetail(dir)); got != 0 {
		t.Errorf("commands while reading = %d, want 0", got)
	}

	// what was read is not read again
	updated, _ := m.Update(previewLoadedMsg{path: "/trash/d", previewResult: previewResult{content: "a/", available: true}})
	updated, _ = updated.Update(sizeLoadedMsg{path: "/trash/d", size: 42})
	model := asModel(t, updated)
	if got := len(model.loadDetail(dir)); got != 0 {
		t.Errorf("commands after reading = %d, want 0", got)
	}
	if got := model.sizeText(dir); got != "42B" {
		t.Errorf("sizeText = %q, want %q", got, "42B")
	}
}

func TestUpdate_PreviewLoadedMsg_AfterMovingOn(t *testing.T) {
	m := newTestModel()
	a := File{File: &trash.File{Name: "a", TrashPath: "/trash/a"}}
	b := File{File: &trash.File{Name: "b", TrashPath: "/trash/b"}}

	updated, _ := m.Update(ShowDetailMsg{file: a})
	updated, _ = updated.Update(ShowDetailMsg{file: b})
	// a's preview comes after the view moved on to b
	updated, _ = updated.Update(previewLoadedMsg{path: "/trash/a", previewResult: previewResult{content: "content of a", available: true}})
	model := asModel(t, updated)

	if got := model.viewport.GetContent(); got == "content of a" {
		t.Error("b's view shows a's preview")
	}
	if _, ok := model.previews["/trash/a"]; !ok {
		t.Error("a's preview is not kept")
	}

	// going back to a shows it without reading it again
	updated, _ = model.Update(ShowDetailMsg{file: a})
	model = asModel(t, updated)
	if got := model.viewport.GetContent(); got != "content of a" {
		t.Errorf("viewport = %q, want a's preview", got)
	}
}

// removeRecorder is a trash that records which files are removed.
type removeRecorder struct {
	trash.Trash
	removed []string
}

func (r *removeRecorder) Remove(file *trash.File) error {
	r.removed = append(r.removed, file.Name)
	return nil
}

func TestUpdate_DetailView_DeleteConfirmsDetailFile(t *testing.T) {
	m := newTestModel()
	file := File{File: &trash.File{Name: "detail.txt"}}
	m.detailFile = file
	m.state.SetView(DetailView)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	model := asModel(t, updated)

	if !model.state.Confirming() {
		t.Fatal("should have opened the confirmation")
	}
	if got := model.state.confirmation.files; len(got) != 1 || got[0] != file {
		t.Errorf("confirmation files = %v, want [detail.txt]", got)
	}
	if model.state.confirmation.state != ConfirmStateYesNo {
		t.Errorf("confirmation state = %v, want %v", model.state.confirmation.state, ConfirmStateYesNo)
	}
}

func TestUpdate_DetailView_DeleteConfirmsSelection(t *testing.T) {
	m := newTestModel()
	f1 := File{File: &trash.File{Name: "a.txt"}}
	f2 := File{File: &trash.File{Name: "b.txt"}}
	m.selection.Add(f1)
	m.selection.Add(f2)
	m.detailFile = File{File: &trash.File{Name: "detail.txt"}}
	m.state.SetView(DetailView)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	model := asModel(t, updated)

	if got := model.state.confirmation.files; len(got) != 2 || got[0] != f1 || got[1] != f2 {
		t.Errorf("confirmation files = %v, want [a.txt b.txt]", got)
	}
	if model.state.confirmation.state != ConfirmStateTypeYES {
		t.Errorf("confirmation state = %v, want %v", model.state.confirmation.state, ConfirmStateTypeYES)
	}
}

func TestUpdate_ConfirmView_DeletesConfirmedFiles(t *testing.T) {
	m := newTestModel()
	rec := &removeRecorder{}
	m.trash = rec
	confirmed := File{File: &trash.File{Name: "confirmed.txt"}}
	other := File{File: &trash.File{Name: "other.txt"}}
	m.list.setFiles([]File{other}, now())
	m.selection.Add(confirmed)
	m.state.SetView(ListView)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	m = asModel(t, updated)
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	model := asModel(t, updated)

	if cmd == nil {
		t.Fatal("should return the delete cmd")
	}
	cmd()
	if len(rec.removed) != 1 || rec.removed[0] != "confirmed.txt" {
		t.Errorf("removed = %v, want [confirmed.txt]", rec.removed)
	}
	if model.selection.Contains(confirmed) {
		t.Error("deleted file should be dropped from the selection")
	}
}

func TestUpdate_ConfirmView_TypeYES_DeletesConfirmedFiles(t *testing.T) {
	m := newTestModel()
	rec := &removeRecorder{}
	m.trash = rec
	f1 := File{File: &trash.File{Name: "a.txt"}}
	f2 := File{File: &trash.File{Name: "b.txt"}}
	kept := File{File: &trash.File{Name: "kept.txt"}}
	m.list.setFiles([]File{kept}, now())
	m.selection.Add(f1)
	m.selection.Add(f2)
	m.state.SetView(ListView)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	m = asModel(t, updated)
	for _, r := range "YES" {
		updated, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = asModel(t, updated)
	}
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model := asModel(t, updated)

	if cmd == nil {
		t.Fatal("should return the delete cmd")
	}
	cmd()
	if len(rec.removed) != 2 || rec.removed[0] != "a.txt" || rec.removed[1] != "b.txt" {
		t.Errorf("removed = %v, want [a.txt b.txt]", rec.removed)
	}
	if len(model.selection.items) != 0 {
		t.Errorf("selection = %v, want empty", model.selection.items)
	}
}

func TestUpdate_ListView_DeleteSelectionWithoutCursor(t *testing.T) {
	m := newTestModel()
	f1 := File{File: &trash.File{Name: "a.txt"}}
	m.selection.Add(f1)
	m.state.SetView(ListView)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	model := asModel(t, updated)

	if !model.state.Confirming() {
		t.Fatal("should have opened the confirmation")
	}
	if got := model.state.confirmation.files; len(got) != 1 || got[0] != f1 {
		t.Errorf("confirmation files = %v, want [a.txt]", got)
	}
}

// Deleting from the detail view with files selected used to keep the yes/no
// state, so pressing Y deleted the file on the cursor instead of the
// selection the dialog was opened for.
func TestUpdate_DetailView_DeleteRemovesSelectionNotCursor(t *testing.T) {
	m := newTestModel()
	rec := &removeRecorder{}
	m.trash = rec
	f1 := File{File: &trash.File{Name: "a.txt"}}
	f2 := File{File: &trash.File{Name: "b.txt"}}
	cursor := File{File: &trash.File{Name: "cursor.txt"}}
	m.list.setFiles([]File{cursor}, now())
	m.selection.Add(f1)
	m.selection.Add(f2)
	m.detailFile = cursor
	m.state.SetView(DetailView)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'D', Text: "D"})
	m = asModel(t, updated)
	var cmd tea.Cmd
	for _, msg := range []tea.KeyPressMsg{
		{Code: 'Y', Text: "Y"},
		{Code: 'E', Text: "E"},
		{Code: 'S', Text: "S"},
		{Code: tea.KeyEnter},
	} {
		if cmd != nil {
			break
		}
		updated, cmd = m.Update(msg)
		m = asModel(t, updated)
	}

	if cmd == nil {
		t.Fatal("should return the delete cmd")
	}
	cmd()
	if len(rec.removed) != 2 || rec.removed[0] != "a.txt" || rec.removed[1] != "b.txt" {
		t.Errorf("removed = %v, want [a.txt b.txt]", rec.removed)
	}
}

func TestLoadPreviewCmd_DirectoryGone(t *testing.T) {
	// the directory is deleted before its preview is read
	f := File{File: &trash.File{Name: "gone", TrashPath: filepath.Join(t.TempDir(), "gone"), IsDir: true}}
	msg, ok := loadPreviewCmd(f, 60, 20)().(previewLoadedMsg)
	if !ok {
		t.Fatalf("got %T, want previewLoadedMsg", msg)
	}
	if msg.available {
		t.Error("a deleted directory has a preview")
	}
}

func TestUpdate_Filter_CtrlCQuits(t *testing.T) {
	m := newTestModel()
	m.list.filtering = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if asModel(t, updated).state.current != Quitting || cmd == nil {
		t.Error("ctrl+c while filtering does not quit")
	}
}

func TestUpdate_HeightFromCursor(t *testing.T) {
	m := newTestModel()
	m.settled = false
	m.cursorY = -1
	if v := m.View().Content; v != "" {
		t.Errorf("drawn before the cursor is known:\n%s", v)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	updated, _ = updated.Update(tea.CursorPositionMsg{Y: 4})
	m = asModel(t, updated)
	if !m.settled || m.height != 36 {
		t.Errorf("settled %v, height %d; want true, 36 (the rows below the prompt)", m.settled, m.height)
	}
}

func TestTargets(t *testing.T) {
	m := newTestModel()
	a, b := listFile("a", time.Hour, "run"), listFile("b", time.Hour+time.Second, "run")
	c := listFile("c", 2*time.Hour, "")
	m.list.setFiles([]File{a, b, c}, listNow)
	m.list.setView(viewBatches, listNow)
	m.list.toggle(false, listNow) // close the run's fold, the cursor on it

	// on a fold: its files
	if got := m.targets(listNow); len(got) != 2 {
		t.Errorf("targets on a fold = %d files, want 2", len(got))
	}
	// the selection wins wherever the cursor is
	m.selection.Add(c)
	if got := m.targets(listNow); len(got) != 1 || got[0].Name != "c" {
		t.Errorf("targets with a selection = %v, want [c]", got)
	}
}

func TestNextView(t *testing.T) {
	m := newTestModel()
	m.config.Views = []string{"batches", "recent"}
	m.list.view = viewRecent
	if got := m.nextView(); got != "batches" {
		t.Errorf("nextView after recent = %q, want batches (the views wrap)", got)
	}
}

// Moving from a file to its fold before the file's preview is due must
// drop the read: the preview then shows no file.
func TestUpdate_PreviewDueAfterMovingToFold(t *testing.T) {
	m := newTestModel()
	m.width = 140 // the preview shows right of the list
	a, b := listFile("a", time.Hour, "run"), listFile("b", time.Hour+time.Second, "run")
	m.list.setFiles([]File{a, b}, listNow)
	m.list.setView(viewBatches, listNow)
	m.list.move(1, listNow) // on a, in the open fold
	m, _ = m.followCursor(nil)
	due := previewDueMsg{seq: m.previewSeq}

	m.list.toggle(false, listNow) // close the fold, the cursor on it
	m, _ = m.followCursor(nil)
	updated, cmd := m.Update(due) // must not panic
	if cmd != nil {
		t.Errorf("a read started for the file the cursor left")
	}
	_ = updated
}

func TestUpdate_PickerEscKeepsQuery(t *testing.T) {
	m := newTestModel()
	m.list.setFiles([]File{listFile("a", time.Hour, "")}, listNow)
	m.list.setView(viewPicker, listNow)
	m.list.filtering = true
	m.list.filter.SetValue("a")
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = asModel(t, updated)
	if m.list.filtering || m.list.filter.Value() != "a" {
		t.Errorf("after esc: filtering %v, query %q; want false, a", m.list.filtering, m.list.filter.Value())
	}
	// v now works on the list
	updated, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
	if got := asModel(t, updated).list.view; got == viewPicker {
		t.Error("v did not change the view after esc")
	}
}

func TestUpdate_SortMenu(t *testing.T) {
	m := newTestModel()
	m.list.setView(viewPicker, listNow)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	m = asModel(t, updated)
	if !m.sortMenu {
		t.Fatal("s did not open the sort menu")
	}
	updated, _ = m.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	m = asModel(t, updated)
	if m.sortMenu || m.list.sort != 2 {
		t.Errorf("after 3: menu %v, sort %d; want closed, 2", m.sortMenu, m.list.sort)
	}
}

// A size that reorders the picker by size keeps the cursor, the preview
// and what enter acts on on the same file.
func TestUpdate_SizeKeepsCursorInSizeOrder(t *testing.T) {
	m := newTestModel()
	a, b := listFile("a", time.Hour, ""), listFile("b", 2*time.Hour, "")
	a.IsDir, b.IsDir = true, true
	m.list.setFiles([]File{a, b}, listNow)
	m.list.setView(viewPicker, listNow)
	m.list.sizeOf = m.knownSize
	m.list.sort = sortBySize
	m.sizes["/trash/a"] = 10
	m.list.toStart(listNow) // on a, the only one sized
	updated, _ := m.Update(sizeLoadedMsg{path: "/trash/b", size: 99})
	m = asModel(t, updated)
	if f, _ := m.list.current(listNow); f.Name != "a" {
		t.Errorf("cursor on %q after b was sized, want a", f.Name)
	}
}

// A paste into the filter narrows the list and moves the cursor as typing
// does.
func TestUpdate_PasteNarrows(t *testing.T) {
	m := newTestModel()
	m.list.setFiles([]File{listFile("apple", time.Hour, ""), listFile("berry", 2*time.Hour, "")}, listNow)
	m.list.setView(viewPicker, listNow)
	m.list.filtering = true
	m.list.filter.Focus()
	updated, _ := m.Update(tea.PasteMsg{Content: "berry"})
	m = asModel(t, updated)
	if f, _ := m.list.current(listNow); f.Name != "berry" {
		t.Errorf("cursor on %q after pasting berry, want berry", f.Name)
	}
}

func TestUpdate_PickerHMovesByPage(t *testing.T) {
	m := newTestModel()
	var files []File
	for i := range 5 {
		files = append(files, listFile(fmt.Sprintf("f%d", i), time.Duration(i+1)*time.Hour, ""))
	}
	m.list.setFiles(files, listNow)
	m.list.setView(viewPicker, listNow)
	m.list.shownN = 2
	m.list.toEnd(listNow)
	updated, _ := m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	m = asModel(t, updated)
	// a page up from f4, not back to the top
	if f, _ := m.list.current(listNow); f.Name != "f2" {
		t.Errorf("cursor on %q after h, want f2", f.Name)
	}
}

func TestUpdate_HelpOverlay(t *testing.T) {
	for _, closeKey := range []tea.KeyPressMsg{
		{Code: '?', Text: "?"}, {Code: tea.KeyEscape}, {Code: 'q', Text: "q"},
	} {
		m := newTestModel()
		updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
		m = asModel(t, updated)
		if !m.helpOpen {
			t.Fatal("? did not open the key list")
		}
		// the keys go to the key list, not the view under it
		updated, _ = m.Update(tea.KeyPressMsg{Code: 'v', Text: "v"})
		m = asModel(t, updated)
		if m.list.view != viewRecent {
			t.Error("v changed the view under the key list")
		}
		updated, cmd := m.Update(closeKey)
		m = asModel(t, updated)
		if m.helpOpen || cmd != nil {
			t.Errorf("%s: open %v, cmd %v; want closed, no quit", closeKey.String(), m.helpOpen, cmd != nil)
		}
	}
}

func TestUpdate_ScopeStatusClears(t *testing.T) {
	m := newTestModel()
	m.list.cwd = "/home/user"
	m.list.setFiles([]File{listFile("a", time.Hour, "")}, listNow)
	updated, _ := m.Update(tea.KeyPressMsg{Code: '.', Text: "."})
	m = asModel(t, updated)
	if m.list.scoped || m.status == "" {
		t.Fatalf("after .: scoped %v, status %q; want false and a status", m.list.scoped, m.status)
	}
	first := m.statusSeq
	updated, _ = m.Update(tea.KeyPressMsg{Code: '.', Text: "."})
	m = asModel(t, updated)
	// the first status's timer does not clear the second
	updated, _ = m.Update(statusClearMsg{seq: first})
	if m = asModel(t, updated); m.status == "" {
		t.Error("an older timer cleared the status")
	}
	updated, _ = m.Update(statusClearMsg{seq: m.statusSeq})
	if m = asModel(t, updated); m.status != "" {
		t.Errorf("status %q after its time, want none", m.status)
	}
}

func TestUpdate_CtrlCQuitsOverMenus(t *testing.T) {
	for name, open := range map[string]func(*Model){
		"help": func(m *Model) { m.helpOpen = true },
		"sort": func(m *Model) { m.sortMenu = true },
	} {
		m := newTestModel()
		open(&m)
		updated, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if asModel(t, updated).state.current != Quitting || cmd == nil {
			t.Errorf("%s: ctrl+c did not quit", name)
		}
	}
}
