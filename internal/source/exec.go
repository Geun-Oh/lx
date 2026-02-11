package source

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Geun-Oh/lx/internal/entry"
)

// ExecSource executes a command and streams its stdout/stderr as LogEntry values.
type ExecSource struct {
	command string
	args    []string
	seq     atomic.Uint64
}

// NewExecSource creates a source that runs the given command with arguments.
func NewExecSource(command string, args []string) *ExecSource {
	return &ExecSource{
		command: command,
		args:    args,
	}
}

// Name returns the source identifier.
func (s *ExecSource) Name() string {
	return fmt.Sprintf("exec:%s", s.command)
}

// Start executes the command and returns a channel of log entries.
// The channel is closed when the command exits or ctx is cancelled.
func (s *ExecSource) Start(ctx context.Context) (<-chan entry.LogEntry, error) {
	cmd := exec.CommandContext(ctx, s.command, s.args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	ch := make(chan entry.LogEntry, 256)
	var wg sync.WaitGroup
	wg.Add(2)

	go s.readStream(ctx, "stdout", stdoutPipe, ch, &wg)
	go s.readStream(ctx, "stderr", stderrPipe, ch, &wg)

	go func() {
		wg.Wait()
		_ = cmd.Wait()
		close(ch)
	}()

	return ch, nil
}

// readStream reads lines from a pipe and sends them to the channel.
func (s *ExecSource) readStream(ctx context.Context, stream string, r io.ReadCloser, ch chan<- entry.LogEntry, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(r)
	// Increase buffer size to 1MB for long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		raw := scanner.Bytes()
		// Copy raw bytes to avoid scanner buffer reuse.
		rawCopy := make([]byte, len(raw))
		copy(rawCopy, raw)

		ch <- entry.LogEntry{
			Timestamp: time.Now(),
			Stream:    stream,
			Source:    s.Name(),
			Message:   scanner.Text(),
			Raw:       rawCopy,
			Seq:       s.seq.Add(1),
		}
	}
}
