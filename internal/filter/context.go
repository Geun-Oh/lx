package filter

import (
	"github.com/Geun-Oh/lx/internal/entry"
)

// ContextBuffer provides grep-like --before / --after context lines.
// It wraps a primary filter and buffers entries to emit context around matches.
type ContextBuffer struct {
	filter     Filter
	beforeN    int
	afterN     int
	ringBuf    []entry.LogEntry // circular buffer of recent entries
	ringPos    int
	afterCount int // remaining "after" lines to emit
}

// NewContextBuffer creates a context-aware filter wrapper.
// before is the number of lines before a match to include.
// after is the number of lines after a match to include.
func NewContextBuffer(f Filter, before, after int) *ContextBuffer {
	size := before + 1
	if size < 1 {
		size = 1
	}
	return &ContextBuffer{
		filter:  f,
		beforeN: before,
		afterN:  after,
		ringBuf: make([]entry.LogEntry, size),
	}
}

// Process evaluates an entry and returns entries to emit (including context).
// Returns nil if the entry should not be emitted yet.
func (cb *ContextBuffer) Process(e *entry.LogEntry) []entry.LogEntry {
	isMatch := cb.filter.Match(e)

	// Store in ring buffer.
	cb.ringBuf[cb.ringPos%len(cb.ringBuf)] = *e
	cb.ringPos++

	if isMatch {
		var result []entry.LogEntry

		// Emit "before" context lines from ring buffer.
		start := cb.ringPos - cb.beforeN - 1
		if start < 0 {
			start = 0
		}
		for i := start; i < cb.ringPos-1; i++ {
			idx := i % len(cb.ringBuf)
			result = append(result, cb.ringBuf[idx])
		}

		// Emit the matching line itself.
		result = append(result, *e)

		// Set after counter.
		cb.afterCount = cb.afterN
		return result
	}

	// If we're still in the "after" window, emit this line.
	if cb.afterCount > 0 {
		cb.afterCount--
		return []entry.LogEntry{*e}
	}

	return nil
}

// Name returns the filter description.
func (cb *ContextBuffer) Name() string {
	return "context"
}
