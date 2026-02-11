package source

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/Geun-Oh/lx/internal/entry"
)

// FileSource reads log lines from a file, optionally following new writes (tail -f).
type FileSource struct {
	path   string
	follow bool
	seq    atomic.Uint64
}

// NewFileSource creates a source that reads from a file.
// If follow is true, it continues reading as new lines are appended.
func NewFileSource(path string, follow bool) *FileSource {
	return &FileSource{
		path:   path,
		follow: follow,
	}
}

// Name returns the source identifier.
func (s *FileSource) Name() string {
	return fmt.Sprintf("file:%s", s.path)
}

// Start opens the file and returns a channel of log entries.
func (s *FileSource) Start(ctx context.Context) (<-chan entry.LogEntry, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", s.path, err)
	}

	ch := make(chan entry.LogEntry, 256)

	go func() {
		defer close(ch)
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for {
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
					Stream:    "file",
					Source:    s.Name(),
					Message:   scanner.Text(),
					Raw:       rawCopy,
					Seq:       s.seq.Add(1),
				}
			}

			if !s.follow {
				return
			}

			// Poll for new data when following.
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				// Reset scanner error state and continue reading.
				scanner = bufio.NewScanner(f)
				scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			}
		}
	}()

	return ch, nil
}
