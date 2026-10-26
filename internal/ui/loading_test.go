package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/babarot/gomi/internal/trash"
)

// newLoadingModel is a model reading its files with load, started at start
func newLoadingModel(load Loader, start time.Time) Model {
	m := newTestModel()
	m.load = load
	m.progress = &trash.Progress{}
	m.loadStart = start
	m.state.SetView(Loading)
	return m
}

func TestLoading_KeysOnlyQuit(t *testing.T) {
	m := newLoadingModel(nil, time.Now())

	for _, k := range []tea.KeyPressMsg{
		{Code: 'v', Text: "v"},
		{Code: '/', Text: "/"},
		{Code: tea.KeyEnter},
		{Code: 'j', Text: "j"},
	} {
		updated, cmd := m.Update(k)
		if got := asModel(t, updated).state.current; got != Loading {
			t.Errorf("%s: state = %v, want loading", k, got)
		}
		if cmd != nil {
			t.Errorf("%s: returned a command while loading", k)
		}
	}

	for _, k := range []tea.KeyPressMsg{{Code: 'q', Text: "q"}, {Code: 'c', Mod: tea.ModCtrl}} {
		// the state is shared between copies of a model
		updated, cmd := newLoadingModel(nil, time.Now()).Update(k)
		if got := asModel(t, updated).state.current; got != Quitting {
			t.Errorf("%s: state = %v, want quit", k, got)
		}
		if cmd == nil {
			t.Errorf("%s: should quit", k)
		}
	}
}

// The files arrive all at once; the list then shows them, and deleting
// files lists the rest from them
func TestLoading_FilesLoaded(t *testing.T) {
	m := newLoadingModel(nil, time.Now())
	files := []File{newTestFile("a.txt"), newTestFile("b.txt")}

	updated, _ := m.Update(FileListLoadedMsg{files: files})
	model := asModel(t, updated)
	if model.state.current != ListView {
		t.Errorf("state = %v, want list view", model.state.current)
	}
	if len(model.files) != 2 || len(model.list.files) != 2 {
		t.Errorf("files = %d, list = %d, want 2 and 2", len(model.files), len(model.list.files))
	}

	// the tick stops once the list is shown
	if _, cmd := model.Update(loadTickMsg{}); cmd != nil {
		t.Error("the loading tick goes on after the list is shown")
	}
}

// An error from the Loader ends the UI with nothing drawn, not as an error
// the view shows
func TestLoading_Failed(t *testing.T) {
	m := newLoadingModel(nil, time.Now())
	want := errors.New("cannot read trash")

	updated, cmd := m.Update(loadFailedMsg{err: want})
	model := asModel(t, updated)
	if !errors.Is(model.loadErr, want) {
		t.Errorf("loadErr = %v, want %v", model.loadErr, want)
	}
	if model.err != nil {
		t.Errorf("err = %v, want none, so that the view draws nothing", model.err)
	}
	if model.state.current != Quitting || cmd == nil {
		t.Errorf("state = %v, want quit", model.state.current)
	}
	if v := model.view(); v != "" {
		t.Errorf("view = %q, want nothing", v)
	}
}

func TestLoadCmd_NewestFirst(t *testing.T) {
	day := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	load := func(p *trash.Progress) ([]*trash.File, error) {
		return []*trash.File{
			{Name: "old", DeletedAt: day},
			{Name: "new", DeletedAt: day.Add(time.Hour)},
		}, nil
	}
	m := newLoadingModel(load, time.Now())

	msg, ok := m.loadCmd()().(FileListLoadedMsg)
	if !ok {
		t.Fatalf("loadCmd returned %T, want FileListLoadedMsg", msg)
	}
	if len(msg.files) != 2 || msg.files[0].Name != "new" {
		t.Errorf("files = %v, want new first", msg.files)
	}
}

func TestLoadingView(t *testing.T) {
	defer func(f func() time.Time) { now = f }(now)
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	m := newLoadingModel(nil, start)

	// nothing until the read takes longer than loadingDelay
	now = func() time.Time { return start.Add(loadingDelay - time.Millisecond) }
	if v := m.view(); v != "" {
		t.Errorf("view before the delay = %q, want nothing", v)
	}

	now = func() time.Time { return start.Add(loadingDelay) }
	if v := m.view(); !strings.Contains(v, "Reading the trash") || strings.Contains(v, "/") {
		t.Errorf("view with no count yet = %q, want the spinner alone", v)
	}

	m.progress.AddTotal(1527)
	m.progress.AddRead(673)
	v := m.view()
	if !strings.Contains(v, "673 / 1527") {
		t.Errorf("view = %q, want the count", v)
	}
	if lines := strings.Split(v, "\n"); len(lines) != 2 || lipgloss.Width(lines[1]) != 2+loadingBarWidth {
		t.Errorf("view = %q, want the count and a bar %d wide under it", v, loadingBarWidth)
	}
}

// With nothing to show, the UI ends only once the terminal has answered
// what it asked at the start, so that no answer is left for the shell
func TestLoading_FailedWaitsForAnswers(t *testing.T) {
	m := newLoadingModel(nil, time.Now())
	m.settled = false

	updated, _ := m.Update(loadFailedMsg{err: errors.New("trash is empty")})
	m = asModel(t, updated)
	if _, cmd := m.Update(tea.CursorPositionMsg{Y: 3}); cmd != nil {
		t.Error("quit before the terminal told its background")
	}

	m.settled = true
	_, cmd := m.Update(tea.BackgroundColorMsg{})
	if cmd == nil {
		t.Fatal("did not quit once the terminal answered")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("the command after the answers is not quit")
	}

	// a terminal that does not answer is waited for only so long
	if _, cmd := m.Update(answerWaitedMsg{}); cmd == nil {
		t.Error("did not quit after the wait")
	}
}
