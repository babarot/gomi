package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ClassicDelegate struct {
	selectionManager *SelectionManager
}

func (h ClassicDelegate) Height() int                               { return 1 }
func (h ClassicDelegate) Spacing() int                              { return 0 }
func (h ClassicDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (h ClassicDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	file, ok := listItem.(File)
	if !ok {
		return
	}
	var str string
	if file.isSelected() {
		str = selectedItemStyle.Render(fmt.Sprintf("%d. %s (%d)", index+1, file.Name, selectionManager.IndexOf(file)+1))
	} else {
		str = fmt.Sprintf("%d. %s", index+1, file.Name)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			var str = []string{"> "}
			return currentItemStyle.Render(append(str, s...)...)
		}
	}

	fmt.Fprint(w, fn(str))
}
