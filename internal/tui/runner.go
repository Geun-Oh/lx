package tui

import (
	"context"
	"fmt"

	"sync"

	"github.com/Geun-Oh/lx/internal/buffer"
	"github.com/Geun-Oh/lx/internal/entry"
	"github.com/Geun-Oh/lx/internal/filter"
	"github.com/Geun-Oh/lx/internal/monitor"
	"github.com/Geun-Oh/lx/internal/parser"
	"github.com/Geun-Oh/lx/internal/source"
	tea "github.com/charmbracelet/bubbletea"
)

// RunConfig holds configuration for the TUI pipeline.
type RunConfig struct {
	Source  source.Source
	Filters *filter.Chain
	Context *filter.ContextBuffer
	Stats   *monitor.Stats
	Rate    *monitor.RateDetector
	Alerts  *monitor.AlertEngine
	RingBuf *buffer.Ring
	Grok    *parser.GrokParser
}

// Run starts the TUI dashboard with a live source pipeline.
// This function blocks until the user quits.
func Run(ctx context.Context, cfg *RunConfig) error {
	// Create a cancellable context to ensure the source is stopped when the TUI exits.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	model := NewModel(cfg.Stats, cfg.Rate, cfg.Alerts, cfg.RingBuf, cfg.Source.Name())
	program := tea.NewProgram(model, tea.WithAltScreen())

	// Start the source and feed entries to the TUI via tea.Program.Send.
	ch, err := cfg.Source.Start(ctx)
	if err != nil {
		return fmt.Errorf("tui: start source: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range ch {
			cfg.Stats.RecordLine()

			// Auto-detect level.
			if e.Level == entry.LevelUnknown {
				e.Level = filter.DetectLevel(e.Message)
			}

			// Parse structured fields via Grok (if configured).
			if cfg.Grok != nil {
				cfg.Grok.Parse(&e)
			}

			// Store in ring buffer.
			if cfg.RingBuf != nil {
				cfg.RingBuf.Push(e)
			}

			// Apply context buffer.
			if cfg.Context != nil {
				entries := cfg.Context.Process(&e)
				for i := range entries {
					cfg.Stats.RecordMatch()
					program.Send(LogMsg(entries[i]))
					cfg.Rate.Record()
					checkAlerts(program, cfg.Alerts, &entries[i])
				}
				continue
			}

			// Apply filter chain.
			if cfg.Filters != nil && cfg.Filters.Len() > 0 {
				if !cfg.Filters.Match(&e) {
					continue
				}
			}

			cfg.Stats.RecordMatch()

			// Track rate and detect spikes.
			if spiking := cfg.Rate.Record(); spiking {
				program.Send(SpikeMsg{Rate: cfg.Rate.CurrentRate()})
			}

			// Check alerts.
			checkAlerts(program, cfg.Alerts, &e)

			// Send to TUI.
			program.Send(LogMsg(e))
		}

		program.Send(DoneMsg{})
	}()

	_, err = program.Run()

	// Ensure source is stopped and consumer finishes.
	cancel()
	wg.Wait()

	return err
}

func checkAlerts(p *tea.Program, alerts *monitor.AlertEngine, e *entry.LogEntry) {
	if alerts == nil {
		return
	}
	triggered := alerts.Check(e)
	if len(triggered) > 0 {
		p.Send(AlertMsg{Rules: triggered, Entry: *e})
	}
}
