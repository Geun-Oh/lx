// Package filter defines the Filter interface and FilterChain for log filtering.
package filter

import (
	"github.com/Geun-Oh/lx/internal/entry"
)

// Filter determines whether a LogEntry matches a filtering criterion.
type Filter interface {
	// Match returns true if the entry passes this filter.
	Match(e *entry.LogEntry) bool

	// Name returns a human-readable description of this filter.
	Name() string
}

// MatchMode controls how multiple filters are combined.
type MatchMode int

const (
	// MatchAny passes if ANY filter matches (OR logic).
	MatchAny MatchMode = iota
	// MatchAll passes only if ALL filters match (AND logic).
	MatchAll
)

// Chain combines multiple filters with a configurable match mode.
type Chain struct {
	filters []Filter
	mode    MatchMode
}

// NewChain creates a FilterChain with the given mode.
func NewChain(mode MatchMode, filters ...Filter) *Chain {
	return &Chain{
		filters: filters,
		mode:    mode,
	}
}

// Add appends a filter to the chain.
func (c *Chain) Add(f Filter) {
	c.filters = append(c.filters, f)
}

// Match evaluates the chain against an entry.
// Returns true if no filters are configured (pass-through).
func (c *Chain) Match(e *entry.LogEntry) bool {
	if len(c.filters) == 0 {
		return true
	}

	switch c.mode {
	case MatchAll:
		for _, f := range c.filters {
			if !f.Match(e) {
				return false
			}
		}
		return true
	default: // MatchAny
		for _, f := range c.filters {
			if f.Match(e) {
				return true
			}
		}
		return false
	}
}

// Name returns a description of the chain.
func (c *Chain) Name() string {
	if c.mode == MatchAll {
		return "FilterChain(AND)"
	}
	return "FilterChain(OR)"
}

// Len returns the number of filters in the chain.
func (c *Chain) Len() int {
	return len(c.filters)
}
