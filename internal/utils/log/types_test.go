package log

import (
	"testing"
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		name  string
		level Level
		want  string
	}{
		{"important level", ImportantLevel, " IMPORTANT "},
		{"debug level", DebugLevel, "debug"},
		{"info level", InfoLevel, "info"},
		{"warn level", WarnLevel, "warn"},
		{"error level", ErrorLevel, "error"},
		{"fatal level", FatalLevel, "fatal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LogLevelString(tt.level); got != tt.want {
				t.Errorf("LogLevelString(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
