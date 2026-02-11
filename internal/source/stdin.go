package source

import (
	"bufio"
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/Geun-Oh/lx/internal/entry"
)

// StdinSource reads log lines from os.Stdin (pipe mode).
type StdinSource struct {
	seq atomic.Uint64
}

// NewStdinSource creates a source that reads from stdin.
func NewStdinSource() *StdinSource {
	return &StdinSource{}
}

// Name returns the source identifier.
func (s *StdinSource) Name() string {
	return "stdin"
}

// Start reads from stdin and returns a channel of log entries.
func (s *StdinSource) Start(ctx context.Context) (<-chan entry.LogEntry, error) {
	ch := make(chan entry.LogEntry, 256)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(os.Stdin)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			raw := scanner.Bytes()
			rawCopy := make([]byte, len(raw))
			copy(rawCopy, raw)

			ch <- entry.LogEntry{
				Timestamp: time.Now(),
				Stream:    "stdin",
				Source:    s.Name(),
				Message:   scanner.Text(),
				Raw:       rawCopy,
				Seq:       s.seq.Add(1),
			}
		}
	}()

	return ch, nil
}
