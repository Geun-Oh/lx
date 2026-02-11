// Package entry defines the core LogEntry type used throughout the lx pipeline.
package entry

import (
	"fmt"
	"time"
)

// Level represents log severity levels.
type Level int

const (
	LevelUnknown Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of a Level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level. Case-insensitive.
func ParseLevel(s string) Level {
	switch s {
	case "DEBUG", "debug", "Debug":
		return LevelDebug
	case "INFO", "info", "Info":
		return LevelInfo
	case "WARN", "warn", "Warn", "WARNING", "warning":
		return LevelWarn
	case "ERROR", "error", "Error", "ERR", "err":
		return LevelError
	case "FATAL", "fatal", "Fatal", "PANIC", "panic":
		return LevelFatal
	default:
		return LevelUnknown
	}
}

// LogEntry is the normalized log message passed through the pipeline.
type LogEntry struct {
	Timestamp time.Time
	Stream    string            // stdout, stderr, file, docker
	Level     Level             // auto-detected or parsed
	Source    string            // source identifier (filename, container, etc.)
	Message   string            // full message text
	Fields    map[string]string // structured fields (for JSON logs)
	Raw       []byte            // original bytes for zero-copy processing
	Seq       uint64            // monotonic sequence number
}

// Format returns a formatted string representation of the entry.
func (e *LogEntry) Format() string {
	ts := e.Timestamp.Format(time.RFC3339)
	if e.Level != LevelUnknown {
		return fmt.Sprintf("[%s][%s][%s]: %s", ts, e.Stream, e.Level, e.Message)
	}
	return fmt.Sprintf("[%s][%s]: %s", ts, e.Stream, e.Message)
}
