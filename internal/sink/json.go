package sink

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Geun-Oh/lx/internal/entry"
)

// jsonEntry is the serialization format for JSON Lines output.
type jsonEntry struct {
	Timestamp string            `json:"timestamp"`
	Stream    string            `json:"stream"`
	Level     string            `json:"level,omitempty"`
	Source    string            `json:"source,omitempty"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// JSONSink writes log entries as JSON Lines (one JSON object per line).
type JSONSink struct {
	w   io.Writer
	enc *json.Encoder
}

// NewJSONSink creates a JSON Lines sink writing to the given writer.
func NewJSONSink(w io.Writer) *JSONSink {
	if w == nil {
		w = os.Stdout
	}
	return &JSONSink{
		w:   w,
		enc: json.NewEncoder(w),
	}
}

// Write serializes a log entry as a single JSON line.
func (s *JSONSink) Write(e *entry.LogEntry) error {
	je := jsonEntry{
		Timestamp: e.Timestamp.Format("2006-01-02T15:04:05.000Z07:00"),
		Stream:    e.Stream,
		Message:   e.Message,
		Source:    e.Source,
	}
	if e.Level != entry.LevelUnknown {
		je.Level = e.Level.String()
	}
	if len(e.Fields) > 0 {
		je.Fields = e.Fields
	}
	return s.enc.Encode(je)
}

// Flush is a no-op for JSON sink.
func (s *JSONSink) Flush() error { return nil }

// Close is a no-op for JSON sink.
func (s *JSONSink) Close() error { return nil }

// Name returns the sink identifier.
func (s *JSONSink) Name() string { return "json" }

// FileSink writes log entries to a file.
type FileSink struct {
	inner Sink
	file  *os.File
}

// NewFileSink creates a sink that writes to the given file path.
// The format parameter selects the inner formatter: "json" or "text" (default).
func NewFileSink(path string, format string) (*FileSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open output file %s: %w", path, err)
	}

	var inner Sink
	switch format {
	case "json":
		inner = NewJSONSink(f)
	default:
		inner = NewTerminalSink(f, false)
	}

	return &FileSink{inner: inner, file: f}, nil
}

// Write delegates to the inner sink.
func (s *FileSink) Write(e *entry.LogEntry) error {
	return s.inner.Write(e)
}

// Flush syncs the file to disk.
func (s *FileSink) Flush() error {
	return s.file.Sync()
}

// Close flushes and closes the file.
func (s *FileSink) Close() error {
	if err := s.Flush(); err != nil {
		return err
	}
	return s.file.Close()
}

// Name returns the sink identifier.
func (s *FileSink) Name() string {
	return "file:" + s.file.Name()
}
