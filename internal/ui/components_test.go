package ui

import "testing"

func TestOnlySpecialChars(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"...", true},
		{"---", true},
		{"___", true},
		{"._-", true},
		{"", true}, // vacuously true - no non-special chars
		{"a", false},
		{".a", false},
		{"file.txt", false},
		{"-name-", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := onlySpecialChars(tt.input); got != tt.want {
				t.Errorf("onlySpecialChars(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		mimeType string
		want     bool
	}{
		{"image/png", true},
		{"image/jpeg", true},
		{"image/gif", true},
		{"text/plain", false},
		{"application/json", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			if got := isImageFile(tt.mimeType); got != tt.want {
				t.Errorf("isImageFile(%q) = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}
