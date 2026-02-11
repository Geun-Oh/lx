// Package monitor provides real-time statistics collection for the pipeline.
package monitor

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Stats collects pipeline processing metrics in a lock-free manner.
type Stats struct {
	totalLines   atomic.Uint64
	matchedLines atomic.Uint64
	startTime    time.Time
}

// NewStats creates a new statistics collector.
func NewStats() *Stats {
	return &Stats{
		startTime: time.Now(),
	}
}

// RecordLine increments the total line counter.
func (s *Stats) RecordLine() {
	s.totalLines.Add(1)
}

// RecordMatch increments the matched line counter.
func (s *Stats) RecordMatch() {
	s.matchedLines.Add(1)
}

// Total returns the total number of processed lines.
func (s *Stats) Total() uint64 {
	return s.totalLines.Load()
}

// Matched returns the total number of matched lines.
func (s *Stats) Matched() uint64 {
	return s.matchedLines.Load()
}

// Elapsed returns the time since monitoring started.
func (s *Stats) Elapsed() time.Duration {
	return time.Since(s.startTime)
}

// Rate returns the current lines per second.
func (s *Stats) Rate() float64 {
	elapsed := s.Elapsed().Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(s.Total()) / elapsed
}

// Summary returns a formatted summary string.
func (s *Stats) Summary() string {
	elapsed := s.Elapsed()
	total := s.Total()
	matched := s.Matched()

	matchRate := float64(0)
	if total > 0 {
		matchRate = float64(matched) / float64(total) * 100
	}

	return fmt.Sprintf(
		"── Summary ──\n"+
			"  Total lines:   %d\n"+
			"  Matched lines: %d (%.1f%%)\n"+
			"  Duration:      %s\n"+
			"  Throughput:    %.0f lines/s\n"+
			"─────────────",
		total, matched, matchRate,
		elapsed.Round(time.Millisecond),
		s.Rate(),
	)
}
