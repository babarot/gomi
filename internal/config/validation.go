package config

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// validateStrategy validates the trash strategy value
func validateStrategy(fl validator.FieldLevel) bool {
	value := strings.ToLower(fl.Field().String())
	return slices.Contains([]string{"auto", "xdg", "legacy"}, value)
}

// validateAllowEmpty allows empty values for optional fields
func validateAllowEmpty(fl validator.FieldLevel) bool {
	str := strings.TrimSpace(fl.Field().String())
	return str == ""
}

// validateSize validates the size format (e.g., "10MB", "1GB")
func validateSize(fl validator.FieldLevel) bool {
	value := strings.ToUpper(fl.Field().String())
	re := regexp.MustCompile(`^\d+(KB|MB|GB|TB|PB)$`)
	return re.MatchString(value)
}

// validateColorCode checks if the field contains a valid hex color code.
func validateColorCode(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	re := regexp.MustCompile(`^#([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)
	return re.MatchString(value)
}

// expandPath expands environment variables and "~" in paths
func expandPath(path string) (string, error) {
	// Expand "~" to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	// Convert to absolute path
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return abs, nil
}

// Deprecation contains metadata about field deprecation
type Deprecation struct {
	DeprecatedAt time.Time
	RemovalDate  time.Time
	Alternative  string
	StrictMode   bool
}

// validateDeprecated implements the deprecated field validation
func validateDeprecated(fl validator.FieldLevel) bool {
	deprecatedInfo := map[string]Deprecation{
		"trash_dir": {
			DeprecatedAt: time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			RemovalDate:  time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			Alternative:  "trash.gomi_dir",
			StrictMode:   true, // make error
		},
	}

	if fl.Field().String() == "" {
		return true
	}

	name := fl.FieldName()
	info, exists := deprecatedInfo[name]
	if !exists {
		printWarningDeprecated(name, nil)
		return true
	}

	if info.StrictMode {
		printErrorDeprecated(name, info)
		return false
	}

	printWarningDeprecated(name, &info)
	return true
}

// validateDirPath is a validation function for directory paths that works on any OS.
// This custom validator was created because the standard "dirpath" validator in go-playground/validator
// incorrectly marks some valid paths as invalid, particularly on Windows. For example:
// - "C:\Users\name\.dir" (marked invalid even though it's a valid Windows path)
// - "C:\Users\name\.dir\" (marked invalid even with trailing separator)
// This implementation properly handles paths across different OS environments without
// these validation issues.
//
// Empty strings are considered invalid.
func validateDirPath(fl validator.FieldLevel) bool {
	path := strings.TrimSpace(fl.Field().String())
	if path == "" {
		return false
	}

	// Normalize (cleanup) the path
	cleanPath := filepath.Clean(path)

	// Check path format
	// Verify that path is maintained after filepath.Clean
	// (If invalid characters are present, they will be modified by Clean)
	if cleanPath != filepath.Clean(filepath.Clean(path)) {
		return false
	}

	// If path exists, verify that it is a directory
	if fi, err := os.Stat(cleanPath); err == nil {
		return fi.IsDir()
	}

	// If path does not exist:
	// Check error type to determine if path format is valid
	_, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist but format is valid
			return true
		}
		if _, ok := err.(*os.PathError); ok {
			// Path error indicates possible OS constraint violation
			return false
		}
	}

	return true
}
