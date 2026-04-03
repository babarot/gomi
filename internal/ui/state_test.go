package ui

import (
	"testing"
	"time"
)

func TestNewViewState(t *testing.T) {
	s := NewViewState()
	if s.current != ListView {
		t.Errorf("current = %v, want ListView", s.current)
	}
	if s.previous != ListView {
		t.Errorf("previous = %v, want ListView", s.previous)
	}
	if s.detail.dateFormat != DateFormatRelative {
		t.Errorf("dateFormat = %v, want DateFormatRelative", s.detail.dateFormat)
	}
	if !s.detail.showOrigin {
		t.Error("showOrigin should be true")
	}
	if !s.preview.available {
		t.Error("preview.available should be true")
	}
	if s.confirmation.state != ConfirmStateYesNo {
		t.Errorf("confirmation.state = %v, want ConfirmStateYesNo", s.confirmation.state)
	}
}

func TestViewState_SetView(t *testing.T) {
	s := NewViewState()

	s.SetView(DetailView)
	if s.current != DetailView {
		t.Errorf("current = %v, want DetailView", s.current)
	}
	if s.previous != ListView {
		t.Errorf("previous = %v, want ListView", s.previous)
	}

	s.SetView(ConfirmView)
	if s.current != ConfirmView {
		t.Errorf("current = %v, want ConfirmView", s.current)
	}
	if s.previous != DetailView {
		t.Errorf("previous = %v, want DetailView", s.previous)
	}
}

func TestViewState_ToggleDateFormat(t *testing.T) {
	s := NewViewState()

	// Default is Relative
	s.ToggleDateFormat()
	if s.detail.dateFormat != DateFormatAbsolute {
		t.Errorf("dateFormat = %v, want Absolute after first toggle", s.detail.dateFormat)
	}

	s.ToggleDateFormat()
	if s.detail.dateFormat != DateFormatRelative {
		t.Errorf("dateFormat = %v, want Relative after second toggle", s.detail.dateFormat)
	}
}

func TestViewState_ToggleOriginPath(t *testing.T) {
	s := NewViewState()

	s.ToggleOriginPath()
	if s.detail.showOrigin {
		t.Error("showOrigin should be false after toggle")
	}

	s.ToggleOriginPath()
	if !s.detail.showOrigin {
		t.Error("showOrigin should be true after second toggle")
	}
}

func TestViewState_FormatDate(t *testing.T) {
	s := NewViewState()
	now := time.Now()

	// Relative format (default)
	relative := s.FormatDate(now)
	if relative == "" {
		t.Error("relative format should not be empty")
	}

	// Absolute format
	s.ToggleDateFormat()
	absolute := s.FormatDate(now)
	if absolute == "" {
		t.Error("absolute format should not be empty")
	}
	// Should contain date components
	expected := now.Format(time.DateTime)
	if absolute != expected {
		t.Errorf("absolute = %q, want %q", absolute, expected)
	}
}

func TestViewState_SetConfirmState(t *testing.T) {
	s := NewViewState()

	files := []File{{}, {}}
	s.SetConfirmState(ConfirmStateTypeYES, files)

	if s.confirmation.state != ConfirmStateTypeYES {
		t.Errorf("state = %v, want ConfirmStateTypeYES", s.confirmation.state)
	}
	if len(s.confirmation.files) != 2 {
		t.Errorf("files = %d, want 2", len(s.confirmation.files))
	}
	if s.confirmation.yesInput != "" {
		t.Error("yesInput should be reset")
	}
}

func TestViewState_UpdateYesInput(t *testing.T) {
	s := NewViewState()

	// Correct sequence: Y, E, S
	s.UpdateYesInput("Y")
	if s.confirmation.yesInput != "Y" {
		t.Errorf("yesInput = %q, want %q", s.confirmation.yesInput, "Y")
	}

	s.UpdateYesInput("E")
	if s.confirmation.yesInput != "YE" {
		t.Errorf("yesInput = %q, want %q", s.confirmation.yesInput, "YE")
	}

	s.UpdateYesInput("S")
	if s.confirmation.yesInput != "YES" {
		t.Errorf("yesInput = %q, want %q", s.confirmation.yesInput, "YES")
	}

	if !s.IsYesComplete() {
		t.Error("should be complete after Y-E-S")
	}
}

func TestViewState_UpdateYesInput_WrongChars(t *testing.T) {
	s := NewViewState()

	// Wrong first char
	s.UpdateYesInput("y") // lowercase
	if s.confirmation.yesInput != "" {
		t.Errorf("yesInput = %q, want empty (lowercase y rejected)", s.confirmation.yesInput)
	}

	// Start correctly, then wrong
	s.UpdateYesInput("Y")
	s.UpdateYesInput("X") // wrong
	if s.confirmation.yesInput != "Y" {
		t.Errorf("yesInput = %q, want %q (wrong second char rejected)", s.confirmation.yesInput, "Y")
	}
}

func TestViewState_BackspaceYesInput(t *testing.T) {
	s := NewViewState()

	s.UpdateYesInput("Y")
	s.UpdateYesInput("E")
	s.BackspaceYesInput()
	if s.confirmation.yesInput != "Y" {
		t.Errorf("yesInput = %q, want %q after backspace", s.confirmation.yesInput, "Y")
	}

	// Backspace on empty should be safe
	s.BackspaceYesInput()
	s.BackspaceYesInput() // now empty
	s.BackspaceYesInput() // extra backspace on empty
	if s.confirmation.yesInput != "" {
		t.Errorf("yesInput = %q, want empty", s.confirmation.yesInput)
	}
}

func TestViewState_ClearYesInput(t *testing.T) {
	s := NewViewState()
	s.UpdateYesInput("Y")
	s.UpdateYesInput("E")
	s.ClearYesInput()
	if s.confirmation.yesInput != "" {
		t.Error("yesInput should be empty after clear")
	}
}

func TestViewState_IsYesComplete(t *testing.T) {
	s := NewViewState()

	if s.IsYesComplete() {
		t.Error("should not be complete initially")
	}

	s.UpdateYesInput("Y")
	if s.IsYesComplete() {
		t.Error("should not be complete after only Y")
	}

	s.UpdateYesInput("E")
	s.UpdateYesInput("S")
	if !s.IsYesComplete() {
		t.Error("should be complete after YES")
	}
}

func TestViewType_String(t *testing.T) {
	tests := []struct {
		v    ViewType
		want string
	}{
		{ListView, "list view"},
		{DetailView, "detail view"},
		{ConfirmView, "confirm view"},
		{Quitting, "quit"},
		{ViewType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("ViewType(%d).String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}
