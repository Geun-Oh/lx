package filter

import (
	"strings"

	"github.com/Geun-Oh/lx/internal/entry"
)

// KeywordFilter matches entries containing a specific keyword.
// Uses bytes.Contains for zero-copy matching when Raw is available.
type KeywordFilter struct {
	keyword string
}

// NewKeywordFilter creates a filter that matches entries containing the keyword.
func NewKeywordFilter(keyword string) *KeywordFilter {
	return &KeywordFilter{keyword: keyword}
}

// Match returns true if the entry message contains the keyword.
func (f *KeywordFilter) Match(e *entry.LogEntry) bool {
	return strings.Contains(e.Message, f.keyword)
}

// Name returns the filter description.
func (f *KeywordFilter) Name() string {
	return "keyword:" + f.keyword
}
