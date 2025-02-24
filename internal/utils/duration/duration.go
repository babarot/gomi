package duration

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var unitMap = map[string]string{
	"h":      "h",
	"hour":   "h",
	"hours":  "h",
	"d":      "d",
	"day":    "d",
	"days":   "d",
	"w":      "w",
	"week":   "w",
	"weeks":  "w",
	"m":      "m",
	"month":  "m",
	"months": "m",
	"y":      "y",
	"year":   "y",
	"years":  "y",
}

var unitDurations = map[string]time.Duration{
	"h": 1 * time.Hour,
	"d": 24 * time.Hour,
	"w": 7 * 24 * time.Hour,
	"m": 30 * 24 * time.Hour,
	"y": 365 * 24 * time.Hour,
}

var (
	// ErrInvalidFormat indicates the input duration string contains invalid characters
	ErrInvalidFormat = errors.New("invalid duration format")

	// ErrInvalidNumber indicates the numeric part is invalid or not positive
	ErrInvalidNumber = errors.New("invalid duration number")

	// ErrInvalidUnit indicates the unit part is not recognized
	ErrInvalidUnit = errors.New("invalid duration unit")
)

func Parse(input string) (time.Duration, error) {
	if input = strings.TrimSpace(input); input == "" {
		return 0, fmt.Errorf("%w: empty input", ErrInvalidFormat)
	}

	numStr, unit, err := splitNumberAndUnit(strings.ToLower(input))
	if err != nil {
		return 0, fmt.Errorf("%w: invalid characters", ErrInvalidFormat)
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("%w: must be a number", ErrInvalidNumber)
	}

	if num < 0 {
		return 0, fmt.Errorf("%w: must be positive", ErrInvalidNumber)
	}

	if unit == "" {
		return 0, fmt.Errorf("%w: missing unit", ErrInvalidFormat)
	}

	mappedUnit, exists := unitMap[unit]
	if !exists {
		return 0, fmt.Errorf("%w: '%s' (supported: h, d, w, m, y)", ErrInvalidUnit, unit)
	}

	return time.Duration(num) * unitDurations[mappedUnit], nil
}

func splitNumberAndUnit(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("%w: empty input", ErrInvalidFormat)
	}

	numPart := strings.Builder{}
	unitPart := strings.Builder{}

	for _, r := range input {
		switch {
		case unicode.IsDigit(r):
			numPart.WriteRune(r)
		case unicode.IsLetter(r):
			unitPart.WriteRune(r)
		default:
			return "", "", ErrInvalidFormat
		}
	}
	return numPart.String(), unitPart.String(), nil
}
