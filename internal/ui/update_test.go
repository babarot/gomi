package ui

import (
	"errors"
	"testing"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
	"github.com/babarot/gomi/internal/ui/keys"
	"github.com/babarot/gomi/internal/ui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel creates a minimal Model for testing Update logic
func newTestModel() Model {
	cfg := config.NewDefaultConfig()
	km := keys.NewKeyMap(keys.KeyMapConfig{DeleteEnabled: true})

	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 80, 40)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()

	return Model{
		state:    NewViewState(),
		keyMap:   km,
		config:   cfg.UI,
		list:     l,
		viewport: viewport.Model{},
		styles:   styles.New(cfg.UI),
		help:     help.New(),
	}
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := newTestModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	if model.list.Width() != 120 {
		t.Errorf("list width = %d, want 120", model.list.Width())
	}
}

func TestUpdate_ErrorMsg(t *testing.T) {
	m := newTestModel()

	msg := errorMsg{err: errors.New("test error")}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

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
	msg := FileListLoadedMsg{files: []list.Item{file}}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	items := model.list.Items()
	if len(items) != 1 {
		t.Errorf("items = %d, want 1", len(items))
	}
}

func TestUpdate_FileListLoadedMsg_Error(t *testing.T) {
	m := newTestModel()

	msg := FileListLoadedMsg{err: errors.New("load failed")}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

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

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

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

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

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

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	if model.state.current != ListView {
		t.Errorf("state = %v, want ListView", model.state.current)
	}
}

func TestUpdate_DetailView_Quit(t *testing.T) {
	m := newTestModel()
	m.state.SetView(DetailView)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_DetailView_AtSignToggles(t *testing.T) {
	m := newTestModel()
	m.state.SetView(DetailView)

	// Initial state
	initialDateFormat := m.state.detail.dateFormat
	initialShowOrigin := m.state.detail.showOrigin

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("@")}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	if model.state.detail.dateFormat == initialDateFormat {
		t.Error("date format should have toggled")
	}
	if model.state.detail.showOrigin == initialShowOrigin {
		t.Error("showOrigin should have toggled")
	}
}

func TestUpdate_ConfirmView_YesNo_No(t *testing.T) {
	m := newTestModel()
	// Reset global selection manager for test isolation
	selectionManager = &SelectionManager{items: []File{}}

	m.state.SetView(ConfirmView)
	m.state.SetConfirmState(ConfirmStateYesNo, []File{})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	// Should go back to previous view
	if model.state.current == ConfirmView {
		t.Error("should have left ConfirmView after pressing 'n'")
	}
}

func TestUpdate_ConfirmView_YesNo_Quit(t *testing.T) {
	m := newTestModel()
	selectionManager = &SelectionManager{items: []File{}}

	m.state.SetView(ConfirmView)
	m.state.SetConfirmState(ConfirmStateYesNo, []File{})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	if model.state.current != Quitting {
		t.Errorf("state = %v, want Quitting", model.state.current)
	}
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestUpdate_ConfirmView_TypeYES_Backspace(t *testing.T) {
	m := newTestModel()
	selectionManager = &SelectionManager{items: []File{}}

	m.state.SetView(ConfirmView)
	m.state.SetConfirmState(ConfirmStateTypeYES, []File{})

	// Type "Y"
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")}
	updated, _ := m.Update(msg)
	m = updated.(Model)

	if m.state.confirmation.yesInput != "Y" {
		t.Errorf("yesInput = %q, want %q", m.state.confirmation.yesInput, "Y")
	}

	// Backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updated, _ = m.Update(msg)
	m = updated.(Model)

	if m.state.confirmation.yesInput != "" {
		t.Errorf("yesInput = %q, want empty after backspace", m.state.confirmation.yesInput)
	}
}

func TestUpdate_ConfirmView_TypeYES_Esc(t *testing.T) {
	m := newTestModel()
	selectionManager = &SelectionManager{items: []File{}}

	m.state.SetView(ConfirmView)
	m.state.SetConfirmState(ConfirmStateTypeYES, []File{})

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	if model.state.current == ConfirmView {
		t.Error("esc should leave ConfirmView")
	}
}

func TestUpdate_ListView_Enter_NoFiles(t *testing.T) {
	m := newTestModel()
	selectionManager = &SelectionManager{items: []File{}}
	m.state.SetView(ListView)

	// Add an item to the list so SelectedItem works
	f := newTestFile("test.txt")
	m.list.SetItems([]list.Item{f})

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

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
	selectionManager = &SelectionManager{items: []File{f1, f2}}
	m.state.SetView(ListView)
	m.list.SetItems([]list.Item{f1, f2})

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

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
	model := updated.(Model)

	if model.state.current != DetailView {
		t.Errorf("state = %v, want DetailView", model.state.current)
	}
	if model.detailFile.Name != "detail.txt" {
		t.Errorf("detailFile.Name = %q, want %q", model.detailFile.Name, "detail.txt")
	}
}
