package config

import (
	"runtime"
	"testing"

	"github.com/go-playground/validator/v10"
)

// helper to create a validator and test a custom validation function
func testValidation(t *testing.T, tag string, fn validator.Func, value string, want bool) {
	t.Helper()

	validate := validator.New()
	if err := validate.RegisterValidation(tag, fn); err != nil {
		t.Fatal(err)
	}

	type testStruct struct {
		Field string `validate:"test"`
	}

	err := validate.Struct(testStruct{Field: value})
	got := err == nil
	if got != want {
		t.Errorf("validate(%q) = %v, want %v", value, got, want)
	}
}

func TestValidateStrategy(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"auto", true},
		{"xdg", true},
		{"legacy", true},
		{"AUTO", true},
		{"XDG", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			testValidation(t, "test", validateStrategy, tt.value, tt.want)
		})
	}
}

func TestValidateAllowEmpty(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"non-empty", "something", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testValidation(t, "test", validateAllowEmpty, tt.value, tt.want)
		})
	}
}

func TestValidateSize(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"10KB", true},
		{"100MB", true},
		{"1GB", true},
		{"5TB", true},
		{"2PB", true},
		{"10kb", true}, // case insensitive
		{"10mb", true},
		{"10", false},
		{"MB", false},
		{"10BB", false},
		{"", false},
		{"abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			testValidation(t, "test", validateSize, tt.value, tt.want)
		})
	}
}

func TestValidateColorCode(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"#FFF", true},
		{"#fff", true},
		{"#FFFFFF", true},
		{"#ffffff", true},
		{"#A1B2C3", true},
		{"#abc", true},
		{"FFF", false},
		{"#FFFF", false},
		{"#GGGGGG", false},
		{"", false},
		{"red", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			testValidation(t, "test", validateColorCode, tt.value, tt.want)
		})
	}
}

func TestValidateDirPath(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"absolute path", "/tmp/test", true},
		{"home relative", "~/Documents", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"current dir", ".", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testValidation(t, "test", validateDirPath, tt.value, tt.want)
		})
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		{
			name:  "absolute path stays absolute",
			input: "/tmp/test",
			check: func(t *testing.T, result string) {
				if runtime.GOOS == "windows" {
					t.Skip("Unix-specific test")
				}
				if result != "/tmp/test" {
					t.Errorf("got %q, want /tmp/test", result)
				}
			},
		},
		{
			name:  "tilde expansion",
			input: "~/Documents",
			check: func(t *testing.T, result string) {
				if result == "~/Documents" {
					t.Error("tilde was not expanded")
				}
			},
		},
		{
			name:  "env var expansion",
			input: "/tmp/$USER",
			check: func(t *testing.T, result string) {
				// Just verify it doesn't error; env var may or may not be set
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}
