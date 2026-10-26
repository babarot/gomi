package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/trash"
)

// Loader reads the files the UI shows, counting into p how far it has got.
// An error ends the UI, and Render returns it.
type Loader func(p *trash.Progress) ([]*trash.File, error)

const (
	// loadingDelay is how long the loading waits before it is drawn, so
	// that a list read at once does not flicker a spinner
	loadingDelay = 200 * time.Millisecond
	// loadingTick is how often the loading is drawn again
	loadingTick = 100 * time.Millisecond
	// loadingBarWidth is the widest the bar under the spinner gets
	loadingBarWidth = 50
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// loadTickMsg draws the loading again while the files are read
type loadTickMsg struct{}

// loadFailedMsg carries the error the Loader returned
type loadFailedMsg struct {
	err error
}

func loadTick() tea.Cmd {
	return tea.Tick(loadingTick, func(time.Time) tea.Msg { return loadTickMsg{} })
}

// newFile is f as the UI shows it, previewed as cfg says
func newFile(f *trash.File, cfg config.UI) File {
	return File{
		File:            f,
		dirListCommand:  cfg.Preview.DirectoryCommand,
		syntaxHighlight: cfg.Preview.SyntaxHighlight,
		colorscheme:     cfg.Preview.Colorscheme,
	}
}

// loadCmd reads the files with the Loader in the background, and returns
// them all at once, newest first. The list is shown only when every file
// is read, so that a file read late does not move the rows under the
// cursor.
func (m Model) loadCmd() tea.Cmd {
	load, p, cfg := m.load, m.progress, m.config
	return func() tea.Msg {
		files, err := load(p)
		if err != nil {
			return loadFailedMsg{err: err}
		}
		list := make([]File, 0, len(files))
		for _, f := range files {
			list = append(list, newFile(f, cfg))
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].DeletedAt.After(list[j].DeletedAt)
		})
		return FileListLoadedMsg{files: list}
	}
}

// loadingView is the spinner right under the command line, with how many
// of the trash's items are read and a bar of it; nothing is drawn until
// loadingDelay has passed
func (m Model) loadingView() string {
	elapsed := now().Sub(m.loadStart)
	if elapsed < loadingDelay {
		return ""
	}
	frame := spinnerFrames[int(elapsed/loadingTick)%len(spinnerFrames)]
	line := "  " + m.fg(m.palette.Cursor).Render(frame) + " " + m.fg(m.palette.Name).Render("Reading the trash")

	read, total := m.progress.Counts()
	if total == 0 {
		return line
	}
	line += m.fg(m.palette.Description).Render(fmt.Sprintf("   %d / %d", read, total))
	width := min(loadingBarWidth, max(m.width-4, 0))
	done := width * min(read, total) / total
	bar := m.fg(m.palette.Cursor).Render(strings.Repeat("━", done)) +
		m.fg(m.palette.DimmedDescription).Render(strings.Repeat("━", width-done))
	return line + "\n  " + bar
}

// answerWaitedMsg ends the wait for the terminal's answers
type answerWaitedMsg struct{}

// answered reports whether the terminal has answered what Init asked: its
// background, and where the cursor is when the height waits for it
func (m Model) answered() bool {
	return m.bgKnown && m.settled
}

// quitWhenAnswered ends the UI once the terminal has answered what Init
// asked. An answer that comes after the UI is gone is left for the shell
// to read, and shows up at its prompt; it can come late when the files are
// read at once and there is none to show.
func (m Model) quitWhenAnswered() (tea.Model, tea.Cmd) {
	return m, m.quitWhenAnsweredCmd()
}

func (m Model) quitWhenAnsweredCmd() tea.Cmd {
	if m.answered() {
		return tea.Quit
	}
	return nil
}

// waitAnswers ends the UI after cursorWait, for a terminal that does not
// answer
func (m Model) waitAnswers() tea.Cmd {
	return tea.Tick(cursorWait, func(time.Time) tea.Msg { return answerWaitedMsg{} })
}
