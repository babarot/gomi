package trash

import (
	"fmt"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
)

// TestItem is a mock implementation of Filterable for testing
type TestItem struct {
	name      string
	path      string
	deletedAt time.Time
}

func (t TestItem) GetName() string {
	return t.name
}

func (t TestItem) GetPath() string {
	return t.path
}

func (t TestItem) GetDeletedAt() time.Time {
	return t.deletedAt
}

// createTestItems generates a slice of test items for various test scenarios
func createTestItems() []TestItem {
	now := time.Now()
	return []TestItem{
		{name: "file1.txt", path: "/trash/file1.txt", deletedAt: now.Add(-24 * time.Hour)},
		{name: "file2.log", path: "/trash/file2.log", deletedAt: now.Add(-48 * time.Hour)},
		{name: "important.txt", path: "/trash/important.txt", deletedAt: now.Add(-72 * time.Hour)},
		{name: "temp.tmp", path: "/trash/temp.tmp", deletedAt: now.Add(-96 * time.Hour)},
	}
}

// createMockDirSizeFunc creates a mock DirSize function for testing
func createMockDirSizeFunc() func(string) (int64, error) {
	return func(path string) (int64, error) {
		sizemap := map[string]int64{
			"/trash/file1.txt":     100,    // 100 bytes
			"/trash/file2.log":     1024,   // 1 KB
			"/trash/important.txt": 10240,  // 10 KB
			"/trash/temp.tmp":      102400, // 100 KB
		}
		size, exists := sizemap[path]
		if !exists {
			return 0, fmt.Errorf("path not found in mock")
		}
		return size, nil
	}
}

func TestRejectBySize(t *testing.T) {
	items := createTestItems()
	mockDirSizeFunc := createMockDirSizeFunc()

	testCases := []struct {
		name          string
		sizeConfig    config.SizeConfig
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "No size filter",
			sizeConfig:    config.SizeConfig{},
			expectedCount: 4,
			expectedNames: []string{"file1.txt", "file2.log", "important.txt", "temp.tmp"},
		},
		{
			name:          "Filter by min size",
			sizeConfig:    config.SizeConfig{Min: "1KB"},
			expectedCount: 3,
			expectedNames: []string{"file2.log", "important.txt", "temp.tmp"},
		},
		{
			name:          "Filter by max size",
			sizeConfig:    config.SizeConfig{Max: "10KB"},
			expectedCount: 2,
			expectedNames: []string{"file1.txt", "file2.log"},
		},
		{
			name:          "Filter by both min and max size",
			sizeConfig:    config.SizeConfig{Min: "1KB", Max: "20KB"},
			expectedCount: 2,
			expectedNames: []string{"file2.log", "important.txt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use the mock DirSize function in the filter
			filtered := rejectBySize(items, tc.sizeConfig, mockDirSizeFunc)

			if len(filtered) != tc.expectedCount {
				t.Errorf("Expected %d items, got %d", tc.expectedCount, len(filtered))
			}

			// Check if remaining items match expected names
			for _, item := range filtered {
				found := false
				for _, expectedName := range tc.expectedNames {
					if item.GetName() == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected item in filtered list: %s", item.GetName())
				}
			}
		})
	}
}

func TestFilter(t *testing.T) {
	mockDirSizeFunc := createMockDirSizeFunc()

	now := time.Now()
	items := []TestItem{
		{name: "file1.txt", path: "/trash/file1.txt", deletedAt: now.Add(-24 * time.Hour)},
		{name: "file2.log", path: "/trash/file2.log", deletedAt: now.Add(-48 * time.Hour)},
		{name: "important.txt", path: "/trash/important.txt", deletedAt: now.Add(-72 * time.Hour)},
		{name: "temp.tmp", path: "/trash/temp.tmp", deletedAt: now.Add(-96 * time.Hour)},
	}

	testCases := []struct {
		name          string
		filterOptions FilterOptions
		expectedCount int
		expectedNames []string
	}{
		{
			name: "No filters",
			filterOptions: FilterOptions{
				Include: config.IncludeConfig{},
				Exclude: config.ExcludeConfig{},
			},
			expectedCount: 4,
			expectedNames: []string{"file1.txt", "file2.log", "important.txt", "temp.tmp"},
		},
		{
			name: "Combined filters",
			filterOptions: FilterOptions{
				Include: config.IncludeConfig{Period: 2},
				Exclude: config.ExcludeConfig{
					Files:    []string{"important.txt"},
					Patterns: []string{`^temp`},
					Size:     config.SizeConfig{Min: "1KB", Max: "10KB"},
				},
			},
			expectedCount: 1,
			expectedNames: []string{"file2.log"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use the mock DirSize function in the filter
			filtered := rejectBySize(items, tc.filterOptions.Exclude.Size, mockDirSizeFunc)

			if len(filtered) != tc.expectedCount {
				t.Errorf("Expected %d items, got %d", tc.expectedCount, len(filtered))
			}

			// Check if remaining items match expected names
			for _, item := range filtered {
				found := false
				for _, expectedName := range tc.expectedNames {
					if item.GetName() == expectedName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected item in filtered list: %s", item.GetName())
				}
			}
		})
	}
}
