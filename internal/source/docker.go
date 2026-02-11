package source

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/Geun-Oh/lx/internal/entry"
)

// DockerSource reads logs from a Docker container via `docker logs --follow`.
type DockerSource struct {
	container string
	follow    bool
	seq       atomic.Uint64
}

// NewDockerSource creates a source that reads from a Docker container's logs.
func NewDockerSource(container string, follow bool) *DockerSource {
	return &DockerSource{
		container: container,
		follow:    follow,
	}
}

// Name returns the source identifier.
func (s *DockerSource) Name() string {
	return fmt.Sprintf("docker:%s", s.container)
}

// Start executes `docker logs` and returns a channel of log entries.
func (s *DockerSource) Start(ctx context.Context) (<-chan entry.LogEntry, error) {
	args := []string{"logs"}
	if s.follow {
		args = append(args, "--follow")
	}
	args = append(args, "--timestamps", s.container)

	cmd := exec.CommandContext(ctx, "docker", args...)

	// Docker sends stdout and stderr interleaved via stderr when using --follow.
	// Capture both.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("docker stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("docker stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("docker logs start: %w (is docker running?)", err)
	}

	ch := make(chan entry.LogEntry, 256)

	go func() {
		defer close(ch)

		done := make(chan struct{})

		// Read stdout.
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				default:
				}
				ts, msg := parseDockerTimestamp(scanner.Text())
				ch <- entry.LogEntry{
					Timestamp: ts,
					Stream:    "stdout",
					Source:    s.Name(),
					Message:   msg,
					Seq:       s.seq.Add(1),
				}
			}
			done <- struct{}{}
		}()

		// Read stderr.
		go func() {
			scanner := bufio.NewScanner(stderrPipe)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				select {
				case <-ctx.Done():
					return
				default:
				}
				ts, msg := parseDockerTimestamp(scanner.Text())
				ch <- entry.LogEntry{
					Timestamp: ts,
					Stream:    "stderr",
					Source:    s.Name(),
					Message:   msg,
					Seq:       s.seq.Add(1),
				}
			}
			done <- struct{}{}
		}()

		// Wait for both readers.
		<-done
		<-done
		_ = cmd.Wait()
	}()

	return ch, nil
}

// parseDockerTimestamp extracts the timestamp from a Docker log line.
// Docker --timestamps format: "2025-01-26T13:32:19.123456789Z message..."
func parseDockerTimestamp(line string) (time.Time, string) {
	if len(line) < 31 {
		return time.Now(), line
	}

	// Try RFC3339Nano (Docker's format).
	tsStr := line[:30]
	ts, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return time.Now(), line
	}

	msg := line[31:] // skip timestamp + space
	return ts, msg
}
