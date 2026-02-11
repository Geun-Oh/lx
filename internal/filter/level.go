package filter

import (
	"regexp"
	"strings"

	"github.com/Geun-Oh/lx/internal/entry"
)

// levelPatterns maps common log level strings to their entry.Level values.
// Ordered by likelihood of occurrence for early exit.
var levelPatterns = []struct {
	keywords []string
	level    entry.Level
}{
	{[]string{"ERROR", "ERR"}, entry.LevelError},
	{[]string{"WARN", "WARNING"}, entry.LevelWarn},
	{[]string{"INFO"}, entry.LevelInfo},
	{[]string{"DEBUG", "TRACE"}, entry.LevelDebug},
	{[]string{"FATAL", "PANIC", "CRITICAL"}, entry.LevelFatal},
}

// levelRegex detects log levels in common formats like [ERROR], level=error, etc.
var levelRegex = regexp.MustCompile(`(?i)\b(DEBUG|TRACE|INFO|WARN(?:ING)?|ERR(?:OR)?|FATAL|PANIC|CRITICAL)\b`)

// LevelFilter passes only entries at or above the specified severity levels.
type LevelFilter struct {
	allowed map[entry.Level]bool
}

// NewLevelFilter creates a filter that passes entries matching any of the given levels.
// Example: NewLevelFilter(entry.LevelError, entry.LevelWarn)
func NewLevelFilter(levels ...entry.Level) *LevelFilter {
	allowed := make(map[entry.Level]bool, len(levels))
	for _, l := range levels {
		allowed[l] = true
	}
	return &LevelFilter{allowed: allowed}
}

// Match returns true if the entry's level is in the allowed set.
// If the entry has no level set, it attempts auto-detection from the message.
func (f *LevelFilter) Match(e *entry.LogEntry) bool {
	level := e.Level
	if level == entry.LevelUnknown {
		level = DetectLevel(e.Message)
		e.Level = level // cache the detected level
	}
	return f.allowed[level]
}

// Name returns the filter description.
func (f *LevelFilter) Name() string {
	var levels []string
	for l := range f.allowed {
		levels = append(levels, l.String())
	}
	return "level:" + strings.Join(levels, ",")
}

// DetectLevel attempts to extract a log level from a message string.
func DetectLevel(msg string) entry.Level {
	match := levelRegex.FindString(msg)
	if match == "" {
		return entry.LevelUnknown
	}

	upper := strings.ToUpper(match)
	for _, p := range levelPatterns {
		for _, kw := range p.keywords {
			if upper == kw {
				return p.level
			}
		}
	}
	return entry.LevelUnknown
}
