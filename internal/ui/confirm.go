package ui

import (
	"log/slog"

	"github.com/babarot/gomi/internal/ui/confirm"
	tea "github.com/charmbracelet/bubbletea"
)

func Confirm(prompt string) bool {
	m := confirm.New()
	m.Prompt = prompt
	m.DefaultValue = confirm.Denied
	m.Immediately = true

	p := tea.NewProgram(&m)
	if _, err := p.Run(); err != nil {
		slog.Error("confirm failed", "error", err)
		return false
	}

	return m.Selected().IsAccepted()
}
