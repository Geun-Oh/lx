// Package source defines the Source interface and common utilities for log input.
package source

import (
	"context"

	"github.com/Geun-Oh/lx/internal/entry"
)

// Source reads log data from an input and emits LogEntry values on a channel.
// Implementations must close the returned channel when the source is exhausted
// or the context is cancelled.
type Source interface {
	// Start begins reading from the source. The returned channel will receive
	// entries until the source is exhausted or ctx is cancelled.
	// The implementation must close the channel when done.
	Start(ctx context.Context) (<-chan entry.LogEntry, error)

	// Name returns a human-readable identifier for this source.
	Name() string
}
