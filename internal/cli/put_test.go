package cli

import "testing"

func TestIsUnsafePath(t *testing.T) {
	tests := []struct {
		path    string
		unsafe  bool
		wantErr bool
	}{
		{".", true, false},                 // original dot
		{"..", true, false},                // original double dot
		{"./", true, false},                // dot with slash
		{"./.", true, false},               // multiple dots
		{"./../../foo/../..", true, false}, // complex path to root
		{"/", true, false},                 // root
		{"//", true, false},                // double slash
		{"//foo", true, false},             // path with double slash
		{"/foo", false, false},             // normal absolute path
		{"foo", false, false},              // normal relative path
		{"foo/bar", false, false},          // normal nested path
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			unsafe, err := isUnsafePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isUnsafePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if unsafe != tt.unsafe {
				t.Errorf("isUnsafePath() = %v, want %v", unsafe, tt.unsafe)
			}
		})
	}
}
