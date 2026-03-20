package config

import (
	"testing"
)

func TestIsStyleRenderEffectivelyEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"ansi codes only", "\x1b[31m\x1b[0m", true},
		{"ansi with spaces", "\x1b[31m  \x1b[0m", true},
		{"has content", "\x1b[31mhello\x1b[0m", false},
		{"plain text", "hello", false},
		{"dot", ".", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isStyleRenderEffectivelyEmpty(tt.input); got != tt.want {
				t.Errorf("isStyleRenderEffectivelyEmpty(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterEmptyStyledStrings(t *testing.T) {
	input := []string{
		"hello",
		"",
		"\x1b[31m\x1b[0m",
		"world",
		"   ",
	}

	result := filterEmptyStyledStrings(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d: %v", len(result), result)
	}
	if result[0] != "hello" || result[1] != "world" {
		t.Errorf("unexpected result: %v", result)
	}
}
