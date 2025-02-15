package config

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

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
