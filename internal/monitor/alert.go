package monitor

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Geun-Oh/lx/internal/entry"
)

// AlertRule defines a pattern that triggers an alert when matched.
type AlertRule struct {
	Name    string
	Pattern *regexp.Regexp
	Count   int // number of times triggered
}

// AlertEngine evaluates log entries against a set of alert rules.
type AlertEngine struct {
	mu    sync.Mutex
	rules []*AlertRule
}

// NewAlertEngine creates an alert engine with the given regex patterns.
func NewAlertEngine(patterns []string) (*AlertEngine, error) {
	engine := &AlertEngine{}
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid alert pattern %q: %w", p, err)
		}
		engine.rules = append(engine.rules, &AlertRule{
			Name:    p,
			Pattern: re,
		})
	}
	return engine, nil
}

// Check evaluates an entry against all rules. Returns matched rule names.
func (e *AlertEngine) Check(entry *entry.LogEntry) []string {
	if len(e.rules) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	var triggered []string
	for _, r := range e.rules {
		if r.Pattern.MatchString(entry.Message) {
			r.Count++
			triggered = append(triggered, r.Name)
		}
	}
	return triggered
}

// Summary returns a formatted summary of alert counts.
func (e *AlertEngine) Summary() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.rules) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("── Alerts ──\n")
	for _, r := range e.rules {
		sb.WriteString(fmt.Sprintf("  %-30s %d hits\n", r.Name, r.Count))
	}
	sb.WriteString("────────────")
	return sb.String()
}

// TotalAlerts returns the total number of alerts triggered.
func (e *AlertEngine) TotalAlerts() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	total := 0
	for _, r := range e.rules {
		total += r.Count
	}
	return total
}
