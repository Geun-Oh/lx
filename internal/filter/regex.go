package filter

import (
	"fmt"
	"regexp"

	"github.com/Geun-Oh/lx/internal/entry"
)

// RegexFilter matches entries against a pre-compiled regular expression.
// The regex is compiled once at construction, eliminating per-line compilation overhead.
type RegexFilter struct {
	pattern string
	re      *regexp.Regexp
}

// NewRegexFilter creates a filter with a pre-compiled regex pattern.
// Returns an error if the pattern is invalid.
func NewRegexFilter(pattern string) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	return &RegexFilter{pattern: pattern, re: re}, nil
}

// Match returns true if the entry message matches the regex.
func (f *RegexFilter) Match(e *entry.LogEntry) bool {
	return f.re.MatchString(e.Message)
}

// Name returns the filter description.
func (f *RegexFilter) Name() string {
	return "regex:" + f.pattern
}
