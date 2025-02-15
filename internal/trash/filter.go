package trash

import (
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/fs"
	"github.com/docker/go-units"
	"github.com/gobwas/glob"
	"github.com/k1LoW/duration"
)

// Filterable defines the interface that trashed files must implement to be filtered
type Filterable interface {
	// GetName returns the original name of the file
	GetName() string
	// GetPath returns the current path in trash
	GetPath() string
	// GetDeletedAt returns when the file was trashed
	GetDeletedAt() time.Time
}

// FilterOptions holds filtering configuration
type FilterOptions struct {
	Include config.IncludeConfig
	Exclude config.ExcludeConfig
}

// Filter applies filtering rules to a slice of items
func Filter[T Filterable](items []T, opts FilterOptions) []T {
	// Filter by filename exclusions
	items = rejectByNames(items, opts.Exclude.Files)

	// Filter by patterns
	items = rejectByPatterns(items, opts.Exclude.Patterns)

	// Filter by globs
	items = rejectByGlobs(items, opts.Exclude.Globs)

	// Filter by size
	items = rejectBySize(items, opts.Exclude.Size)

	// Filter by time period
	items = filterByPeriod(items, opts.Include.Period)

	return items
}

func rejectByNames[T Filterable](items []T, excludeFiles []string) []T {
	if len(excludeFiles) == 0 {
		return items
	}

	var filtered []T
	for _, item := range items {
		excluded := false
		for _, exclude := range excludeFiles {
			if item.GetName() == exclude {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func rejectByPatterns[T Filterable](items []T, patterns []string) []T {
	if len(patterns) == 0 {
		return items
	}

	var filtered []T
	for _, item := range items {
		excluded := false
		for _, pattern := range patterns {
			if matched, err := regexp.MatchString(pattern, item.GetName()); err == nil && matched {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func rejectByGlobs[T Filterable](items []T, globs []string) []T {
	if len(globs) == 0 {
		return items
	}

	var filtered []T
	for _, item := range items {
		excluded := false
		for _, g := range globs {
			if glob.MustCompile(g).Match(item.GetName()) {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func rejectBySize[T Filterable](items []T, size config.SizeConfig) []T {
	var filtered []T
	for _, item := range items {
		dirSize, err := fs.DirSize(item.GetPath())
		if err != nil {
			continue // Skip items we can't size
		}

		include := true
		if size.Min != "" {
			if min, err := units.FromHumanSize(size.Min); err == nil {
				if dirSize <= min {
					include = false
				}
			}
		}
		if size.Max != "" {
			if max, err := units.FromHumanSize(size.Max); err == nil {
				if max <= dirSize {
					include = false
				}
			}
		}
		if include {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterByPeriod[T Filterable](items []T, period int) []T {
	if period <= 0 {
		return items
	}

	d, err := duration.Parse(fmt.Sprintf("%d days", period))
	if err != nil {
		slog.Error("failed to parse duration", "error", err)
		return items
	}

	var filtered []T
	for _, item := range items {
		if time.Since(item.GetDeletedAt()) < d {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
