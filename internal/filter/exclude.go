package filter

import (
	"strings"

	"github.com/Geun-Oh/lx/internal/entry"
)

// ExcludeFilter is a negative filter that rejects entries matching any exclude pattern.
// Wrap with NegateFilter or use directly — Match returns true if the entry should PASS
// (i.e., does NOT contain any excluded pattern).
type ExcludeFilter struct {
	patterns []string
}

// NewExcludeFilter creates a filter that rejects entries containing any of the patterns.
func NewExcludeFilter(patterns ...string) *ExcludeFilter {
	return &ExcludeFilter{patterns: patterns}
}

// Match returns true if the entry does NOT contain any excluded pattern.
func (f *ExcludeFilter) Match(e *entry.LogEntry) bool {
	for _, p := range f.patterns {
		if strings.Contains(e.Message, p) {
			return false
		}
	}
	return true
}

// Name returns the filter description.
func (f *ExcludeFilter) Name() string {
	return "exclude:" + strings.Join(f.patterns, ",")
}
