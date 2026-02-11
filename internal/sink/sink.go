// Package sink defines the Sink interface for pipeline output.
package sink

import (
	"github.com/Geun-Oh/lx/internal/entry"
)

// Sink receives filtered LogEntry values and writes them to an output destination.
type Sink interface {
	// Write outputs a single log entry.
	Write(e *entry.LogEntry) error

	// Flush ensures all buffered output is written.
	Flush() error

	// Close releases resources held by the sink.
	Close() error

	// Name returns a human-readable identifier for this sink.
	Name() string
}
