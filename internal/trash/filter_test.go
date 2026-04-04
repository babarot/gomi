package trash

import (
	"slices"
	"testing"
	"time"

	"github.com/babarot/gomi/internal/config"
)

// TestItem is a mock implementation of Filterable for testing
type TestItem struct {
	name      string
	path      string
	size      int64
	deletedAt time.Time
}

func (t TestItem) GetName() string         { return t.name }
func (t TestItem) GetPath() string         { return t.path }
func (t TestItem) GetDeletedAt() time.Time { return t.deletedAt }
func (t TestItem) GetSize() int64          { return t.size }

// createTestItems generates a slice of test items for various test scenarios
func createTestItems() []TestItem {
	now := time.Now()
	return []TestItem{
		{name: "file1.txt", path: "/trash/file1.txt", size: 100, deletedAt: now.Add(-24 * time.Hour)},
		{name: "file2.log", path: "/trash/file2.log", size: 1024, deletedAt: now.Add(-48 * time.Hour)},
		{name: "important.txt", path: "/trash/important.txt", size: 10240, deletedAt: now.Add(-72 * time.Hour)},
		{name: "temp.tmp", path: "/trash/temp.tmp", size: 102400, deletedAt: now.Add(-96 * time.Hour)},
	}
}

func TestRejectBySize(t *testing.T) {
	items := createTestItems()

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
			name:          "Empty file with min 0KB is included",
			sizeConfig:    config.SizeConfig{Min: "0KB"},
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
			filtered := rejectBySize(items, tc.sizeConfig)

			if len(filtered) != tc.expectedCount {
				t.Errorf("Expected %d items, got %d", tc.expectedCount, len(filtered))
			}

			// Check if remaining items match expected names
			for _, item := range filtered {
				found := slices.Contains(tc.expectedNames, item.GetName())
				if !found {
					t.Errorf("Unexpected item in filtered list: %s", item.GetName())
				}
			}
		})
	}
}

func TestFilter(t *testing.T) {
	now := time.Now()
	items := []TestItem{
		{name: "file1.txt", path: "/trash/file1.txt", size: 100, deletedAt: now.Add(-24 * time.Hour)},
		{name: "file2.log", path: "/trash/file2.log", size: 1024, deletedAt: now.Add(-48 * time.Hour)},
		{name: "important.txt", path: "/trash/important.txt", size: 10240, deletedAt: now.Add(-72 * time.Hour)},
		{name: "temp.tmp", path: "/trash/temp.tmp", size: 102400, deletedAt: now.Add(-96 * time.Hour)},
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
			filtered := rejectBySize(items, tc.filterOptions.Exclude.Size)

			if len(filtered) != tc.expectedCount {
				t.Errorf("Expected %d items, got %d", tc.expectedCount, len(filtered))
			}

			// Check if remaining items match expected names
			for _, item := range filtered {
				found := slices.Contains(tc.expectedNames, item.GetName())
				if !found {
					t.Errorf("Unexpected item in filtered list: %s", item.GetName())
				}
			}
		})
	}
}
