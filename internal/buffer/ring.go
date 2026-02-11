// Package buffer provides memory-managed buffering for the log pipeline.
package buffer

import (
	"sync"

	"github.com/Geun-Oh/lx/internal/entry"
)

// Ring is a fixed-capacity circular buffer for LogEntry values.
// When full, the oldest entries are silently evicted.
// All operations are goroutine-safe.
type Ring struct {
	mu       sync.RWMutex
	entries  []entry.LogEntry
	head     int // next write position
	count    int // current number of entries
	capacity int
	dropped  uint64 // total evicted entries
}

// NewRing creates a ring buffer with the given capacity.
func NewRing(capacity int) *Ring {
	if capacity <= 0 {
		capacity = 1024
	}
	return &Ring{
		entries:  make([]entry.LogEntry, capacity),
		capacity: capacity,
	}
}

// Push adds an entry to the ring buffer. If full, the oldest entry is evicted.
func (r *Ring) Push(e entry.LogEntry) {
	r.mu.Lock()
	r.entries[r.head] = e
	r.head = (r.head + 1) % r.capacity
	if r.count < r.capacity {
		r.count++
	} else {
		r.dropped++
	}
	r.mu.Unlock()
}

// Snapshot returns a copy of all buffered entries in chronological order.
func (r *Ring) Snapshot() []entry.LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]entry.LogEntry, r.count)
	if r.count < r.capacity {
		copy(result, r.entries[:r.count])
	} else {
		// Buffer is full: read from head (oldest) to end, then from start to head.
		start := r.head % r.capacity
		n := copy(result, r.entries[start:])
		copy(result[n:], r.entries[:start])
	}
	return result
}

// Len returns the current number of entries in the buffer.
func (r *Ring) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.count
}

// Dropped returns the total number of evicted entries.
func (r *Ring) Dropped() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.dropped
}

// Cap returns the buffer capacity.
func (r *Ring) Cap() int {
	return r.capacity
}
