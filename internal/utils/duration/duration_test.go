package duration

import (
	"errors"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr error
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "invalid characters",
			input:   "1d!",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "no number",
			input:   "d",
			wantErr: ErrInvalidNumber,
		},
		{
			name:  "zero",
			input: "0d",
			want:  0,
		},
		{
			name:    "negative number",
			input:   "-1d",
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "invalid unit",
			input:   "1x",
			wantErr: ErrInvalidUnit,
		},
		{
			name:  "1 hour",
			input: "1h",
			want:  time.Hour,
		},
		{
			name:  "1 hour (full word)",
			input: "1hour",
			want:  time.Hour,
		},
		{
			name:  "2 hours (plural)",
			input: "2hours",
			want:  2 * time.Hour,
		},
		{
			name:  "1 day",
			input: "1d",
			want:  24 * time.Hour,
		},
		{
			name:  "1 day (full word)",
			input: "1day",
			want:  24 * time.Hour,
		},
		{
			name:  "2 days (plural)",
			input: "2days",
			want:  48 * time.Hour,
		},
		{
			name:  "1 week",
			input: "1w",
			want:  7 * 24 * time.Hour,
		},
		{
			name:  "1 week (full word)",
			input: "1week",
			want:  7 * 24 * time.Hour,
		},
		{
			name:  "2 weeks (plural)",
			input: "2weeks",
			want:  2 * 7 * 24 * time.Hour,
		},
		{
			name:  "1 month",
			input: "1m",
			want:  30 * 24 * time.Hour,
		},
		{
			name:  "1 month (full word)",
			input: "1month",
			want:  30 * 24 * time.Hour,
		},
		{
			name:  "2 months (plural)",
			input: "2months",
			want:  2 * 30 * 24 * time.Hour,
		},
		{
			name:  "1 year",
			input: "1y",
			want:  365 * 24 * time.Hour,
		},
		{
			name:  "1 year (full word)",
			input: "1year",
			want:  365 * 24 * time.Hour,
		},
		{
			name:  "2 years (plural)",
			input: "2years",
			want:  2 * 365 * 24 * time.Hour,
		},
		{
			name:  "mixed case",
			input: "1DaY",
			want:  24 * time.Hour,
		},
		{
			name:  "with spaces",
			input: " 1d ",
			want:  24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Parse(%q) should return error", tt.input)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Parse(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitNumberAndUnit(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNum  string
		wantUnit string
		wantErr  error
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: ErrInvalidFormat,
		},
		{
			name:     "only number",
			input:    "1",
			wantNum:  "1",
			wantUnit: "",
		},
		{
			name:     "only unit",
			input:    "d",
			wantNum:  "",
			wantUnit: "d",
		},
		{
			name:     "valid input",
			input:    "1d",
			wantNum:  "1",
			wantUnit: "d",
		},
		{
			name:     "valid input with full word",
			input:    "1day",
			wantNum:  "1",
			wantUnit: "day",
		},
		{
			name:    "invalid character",
			input:   "1d!",
			wantErr: ErrInvalidFormat,
		},
		{
			name:     "with spaces",
			input:    " 1d ",
			wantNum:  "1",
			wantUnit: "d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNum, gotUnit, err := splitNumberAndUnit(tt.input)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("splitNumberAndUnit(%q) should return error", tt.input)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("splitNumberAndUnit(%q) error = %v, want %v", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("splitNumberAndUnit(%q) unexpected error: %v", tt.input, err)
				return
			}
			if gotNum != tt.wantNum {
				t.Errorf("splitNumberAndUnit(%q) num = %v, want %v", tt.input, gotNum, tt.wantNum)
			}
			if gotUnit != tt.wantUnit {
				t.Errorf("splitNumberAndUnit(%q) unit = %v, want %v", tt.input, gotUnit, tt.wantUnit)
			}
		})
	}
}
