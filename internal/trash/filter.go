package trash

import (
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"time"

	"github.com/docker/go-units"
	"github.com/gobwas/glob"
	"github.com/k1LoW/duration"

	"github.com/babarot/gomi/internal/config"
	"github.com/babarot/gomi/internal/utils/fs"
)

// Filterable defines the interface that trashed files must implement to be filtered
type Filterable interface {
	// GetName returns the original name of the file
	GetName() string
	// GetPath returns the current path in trash
	GetPath() string
	// GetDeletedAt returns when the file was trashed
	GetDeletedAt() time.Time
	// GetSize returns the cached size in bytes, or 0 if unknown
	GetSize() int64
}

// FilterOptions holds filtering configuration
type FilterOptions struct {
	Include config.IncludeConfig
	Exclude config.ExcludeConfig
}

// Filter applies filtering rules to a slice of items
func Filter[T Filterable](items []T, opts FilterOptions) []T {
	slog.Debug("starting filter",
		"len(items)", len(items),
		slog.Group("exclude",
			"files", len(opts.Exclude.Files),
			"patterns", len(opts.Exclude.Patterns),
			"globs", len(opts.Exclude.Globs),
			"size.max", opts.Exclude.Size.Max,
			"size.min", opts.Exclude.Size.Min,
		),
		slog.Group("include",
			"period", opts.Include.Period,
		),
	)

	// Filter by time period first and by size last: an item without a cached
	// size is sized by walking it, so only what the other filters keep is
	// walked
	items = filterByPeriod(items, opts.Include.Period)
	slog.Debug("after period filtering", "len(items)", len(items))

	// Filter by filename exclusions
	items = rejectByNames(items, opts.Exclude.Files)
	slog.Debug("after name filtering", "len(items)", len(items))

	// Filter by patterns
	items = rejectByPatterns(items, opts.Exclude.Patterns)
	slog.Debug("after pattern filtering", "len(items)", len(items))

	// Filter by globs
	items = rejectByGlobs(items, opts.Exclude.Globs)
	slog.Debug("after glob filtering", "len(items)", len(items))

	// Filter by size
	items = rejectBySize(items, opts.Exclude.Size)
	slog.Debug("after size filtering", "len(items)", len(items))

	return items
}

func rejectByNames[T Filterable](items []T, excludeFiles []string) []T {
	if len(excludeFiles) == 0 {
		return items
	}

	var filtered []T
	for _, item := range items {
		excluded := slices.Contains(excludeFiles, item.GetName())
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

// sizeSetter is implemented by items that keep the size the filter walks
// them for, so it is not walked again
type sizeSetter interface {
	SetSize(int64)
}

// measurer is implemented by items that have their own size for the size
// filter, which may be 0, rather than the size of their contents
type measurer interface {
	MeasuredSize() (size int64, ok bool, err error)
}

// filterSize is the size the size filter judges item by
func filterSize[T Filterable](item T) (int64, error) {
	if m, ok := any(item).(measurer); ok {
		if n, ok, err := m.MeasuredSize(); ok {
			return n, err
		}
	}
	// Use cached size from List(); fall back to DirSize for unknown sizes
	if n := item.GetSize(); n != 0 {
		return n, nil
	}
	n, err := fs.DirSize(item.GetPath())
	if err != nil {
		return 0, err
	}
	if s, ok := any(item).(sizeSetter); ok {
		s.SetSize(n)
	}
	return n, nil
}

func rejectBySize[T Filterable](items []T, size config.SizeConfig) []T {
	if size.Min == "" && size.Max == "" {
		return items
	}

	var filtered []T
	for _, item := range items {
		fileSize, err := filterSize(item)
		if err != nil {
			continue // Skip items we can't size
		}

		include := true
		if size.Min != "" {
			if min, err := units.FromHumanSize(size.Min); err == nil {
				if fileSize < min {
					include = false
				}
			}
		}
		if size.Max != "" {
			if max, err := units.FromHumanSize(size.Max); err == nil {
				if max < fileSize {
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
