// Package pipeline orchestrates Source → Filter → Sink processing.
package pipeline

import (
	"context"
	"fmt"

	"github.com/Geun-Oh/lx/internal/buffer"
	"github.com/Geun-Oh/lx/internal/entry"
	"github.com/Geun-Oh/lx/internal/filter"
	"github.com/Geun-Oh/lx/internal/monitor"
	"github.com/Geun-Oh/lx/internal/sink"
	"github.com/Geun-Oh/lx/internal/source"
)

// Config holds pipeline configuration.
type Config struct {
	Source    source.Source
	Filters   *filter.Chain
	Sinks     []sink.Sink
	Context   *filter.ContextBuffer // optional context lines
	Stats     *monitor.Stats
	RingBuf   *buffer.Ring // optional ring buffer for TUI search
	ShowStats bool
}

// Run executes the pipeline: reads from source, filters, and writes to sinks.
// Blocks until the source is exhausted or ctx is cancelled.
func Run(ctx context.Context, cfg *Config) error {
	if cfg.Source == nil {
		return fmt.Errorf("pipeline: source is required")
	}
	if len(cfg.Sinks) == 0 {
		return fmt.Errorf("pipeline: at least one sink is required")
	}

	ch, err := cfg.Source.Start(ctx)
	if err != nil {
		return fmt.Errorf("pipeline: start source: %w", err)
	}

	for e := range ch {
		cfg.Stats.RecordLine()

		// Auto-detect log level if not set.
		if e.Level == entry.LevelUnknown {
			e.Level = filter.DetectLevel(e.Message)
		}

		// Store in ring buffer (if configured).
		if cfg.RingBuf != nil {
			cfg.RingBuf.Push(e)
		}

		// Context lines mode.
		if cfg.Context != nil {
			entries := cfg.Context.Process(&e)
			for i := range entries {
				cfg.Stats.RecordMatch()
				for _, s := range cfg.Sinks {
					if err := s.Write(&entries[i]); err != nil {
						return fmt.Errorf("pipeline: write to %s: %w", s.Name(), err)
					}
				}
			}
			continue
		}

		// Standard filter chain.
		if cfg.Filters != nil && cfg.Filters.Len() > 0 {
			if !cfg.Filters.Match(&e) {
				continue
			}
		}

		cfg.Stats.RecordMatch()

		for _, s := range cfg.Sinks {
			if err := s.Write(&e); err != nil {
				return fmt.Errorf("pipeline: write to %s: %w", s.Name(), err)
			}
		}
	}

	// Flush and close sinks.
	for _, s := range cfg.Sinks {
		_ = s.Flush()
		_ = s.Close()
	}

	// Print summary if requested.
	if cfg.ShowStats {
		fmt.Println()
		fmt.Println(cfg.Stats.Summary())
	}

	return nil
}
