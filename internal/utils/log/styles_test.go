package log

import (
	"testing"
)

func TestDefaultStyles(t *testing.T) {
	Reset()
	defer Reset()

	styles := DefaultStyles()
	if styles == nil {
		t.Fatal("DefaultStyles() returned nil")
	}

	// Calling again should return the same instance
	styles2 := DefaultStyles()
	if styles != styles2 {
		t.Error("DefaultStyles() should return singleton")
	}
}

func TestHighlight(t *testing.T) {
	result := Highlight("test")
	if result == "" {
		t.Error("Highlight() returned empty string")
	}
}

func TestUnderBold(t *testing.T) {
	result := UnderBold("test")
	if result == "" {
		t.Error("UnderBold() returned empty string")
	}
}
