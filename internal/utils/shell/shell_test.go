package shell

import (
	"os"
	"testing"
)

func TestRunCommand(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		wantErrCode    int
		wantOutputFunc func(string) bool
	}{
		{
			name:        "Simple echo command",
			input:       "echo hello world",
			wantErrCode: 0,
			wantOutputFunc: func(output string) bool {
				return output == "hello world\n"
			},
		},
		{
			name:        "Invalid command",
			input:       "nonexistent-command",
			wantErrCode: 127,
			wantOutputFunc: func(output string) bool {
				return output != ""
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, errCode, err := RunCommand(tc.input)

			if err != nil && errCode == 0 {
				t.Errorf("Unexpected error: %v", err)
			}

			if errCode != tc.wantErrCode {
				t.Errorf("Expected error code %d, got %d", tc.wantErrCode, errCode)
			}

			if !tc.wantOutputFunc(output) {
				t.Errorf("Unexpected output: %q", output)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	// Set a test home directory
	testHome := "/test/home/user"
	os.Setenv("HOME", testHome)
	defer os.Unsetenv("HOME")

	testCases := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "Tilde expansion",
			input:    "~/test",
			expected: testHome + "/test",
			wantErr:  false,
		},
		{
			name:     "Single tilde",
			input:    "~",
			expected: testHome,
			wantErr:  false,
		},
		{
			name:     "Environment variable expansion",
			input:    "$HOME/docs",
			expected: testHome + "/docs",
			wantErr:  false,
		},
		{
			name:     "Braced environment variable expansion",
			input:    "${HOME}/docs",
			expected: testHome + "/docs",
			wantErr:  false,
		},
		{
			name:     "Mixed expansions",
			input:    "~/docs/$HOME/test",
			expected: testHome + "/docs/" + testHome + "/test",
			wantErr:  false,
		},
		{
			name:     "Undefined environment variable",
			input:    "$UNDEFINED_VAR/test",
			expected: "/test",
			wantErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ExpandHome(tc.input)

			if tc.wantErr && err == nil {
				t.Errorf("Expected an error, got nil")
			}

			if err != nil && !tc.wantErr {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestIsShellVarChar(t *testing.T) {
	testCases := []struct {
		name     string
		char     byte
		expected bool
	}{
		{"Lowercase letter", 'a', true},
		{"Uppercase letter", 'Z', true},
		{"Digit", '5', true},
		{"Underscore", '_', true},
		{"Space", ' ', false},
		{"Punctuation", '.', false},
		{"Special character", '$', false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isShellVarChar(tc.char)
			if result != tc.expected {
				t.Errorf("For char %q, expected %v, got %v", tc.char, tc.expected, result)
			}
		})
	}
}

// Benchmark for performance testing
func BenchmarkExpandHome(b *testing.B) {
	os.Setenv("HOME", "/home/testuser")
	input := "~/documents/${HOME}/projects"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExpandHome(input)
	}
}
