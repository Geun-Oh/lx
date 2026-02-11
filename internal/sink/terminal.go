package sink

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Geun-Oh/lx/internal/entry"
)

// color ANSI escape codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// TerminalSink writes log entries to a terminal with optional ANSI color.
type TerminalSink struct {
	w     io.Writer
	color bool
}

// NewTerminalSink creates a sink that writes to the given writer.
// If color is true, output will include ANSI color codes based on log level.
func NewTerminalSink(w io.Writer, color bool) *TerminalSink {
	if w == nil {
		w = os.Stdout
	}
	return &TerminalSink{w: w, color: color}
}

// Write outputs a formatted log entry.
func (s *TerminalSink) Write(e *entry.LogEntry) error {
	ts := e.Timestamp.Format(time.RFC3339)

	if !s.color {
		if e.Level != entry.LevelUnknown {
			_, err := fmt.Fprintf(s.w, "[%s][%s][%s]: %s\n", ts, e.Stream, e.Level, e.Message)
			return err
		}
		_, err := fmt.Fprintf(s.w, "[%s][%s]: %s\n", ts, e.Stream, e.Message)
		return err
	}

	// Colorized output.
	levelColor := s.levelColor(e.Level)
	if e.Level != entry.LevelUnknown {
		_, err := fmt.Fprintf(s.w, "%s[%s]%s[%s]%s[%s]%s: %s\n",
			colorGray, ts, colorReset,
			e.Stream,
			levelColor, e.Level, colorReset,
			e.Message,
		)
		return err
	}
	_, err := fmt.Fprintf(s.w, "%s[%s]%s[%s]: %s\n",
		colorGray, ts, colorReset,
		e.Stream,
		e.Message,
	)
	return err
}

// Flush is a no-op for terminal output.
func (s *TerminalSink) Flush() error { return nil }

// Close is a no-op for terminal output.
func (s *TerminalSink) Close() error { return nil }

// Name returns the sink identifier.
func (s *TerminalSink) Name() string { return "terminal" }

func (s *TerminalSink) levelColor(l entry.Level) string {
	switch l {
	case entry.LevelError, entry.LevelFatal:
		return colorBold + colorRed
	case entry.LevelWarn:
		return colorYellow
	case entry.LevelDebug:
		return colorGray
	default:
		return colorCyan
	}
}
