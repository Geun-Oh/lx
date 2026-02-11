package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Geun-Oh/lx/internal/buffer"
	"github.com/Geun-Oh/lx/internal/entry"
	"github.com/Geun-Oh/lx/internal/filter"
	"github.com/Geun-Oh/lx/internal/monitor"
	"github.com/Geun-Oh/lx/internal/pipeline"
	"github.com/Geun-Oh/lx/internal/sink"
	"github.com/Geun-Oh/lx/internal/source"
	"github.com/spf13/cobra"
)

var (
	// Filter flags.
	keywords     []string
	regexPattern string
	levels       []string
	excludes     []string
	matchMode    string

	// Context flags.
	beforeLines int
	afterLines  int

	// I/O flags.
	inputFile  string
	follow     bool
	outputFile string
	format     string
	color      bool

	// Stats flags.
	showStats  bool
	bufferSize int

	rootCmd = &cobra.Command{
		Use:   "lx [flags] [--] <command> [args...]",
		Short: "lx — real-time log monitoring & extraction tool",
		Long: `lx is a lightweight yet powerful real-time log monitoring and extraction tool.
It executes commands (or reads from stdin/files) and filters their output
using keywords, regex, log levels, and more.

Examples:
  lx -k ERROR -- docker compose logs -f
  lx -k ERROR -k WARN --match-mode or -- kubectl logs -f pod-name
  lx -r "status=[45]\d{2}" -- tail -f /var/log/nginx/access.log
  lx --level ERROR,WARN --color -- ./my-app
  kubectl logs -f pod-name | lx -k ERROR
  lx --file /var/log/app.log -k ERROR --follow --stats`,
		SilenceUsage: true,
		RunE:         run,
	}
)

func init() {
	cobra.OnInitialize()

	// Filter flags.
	rootCmd.Flags().StringArrayVarP(&keywords, "keyword", "k", nil, "keyword filter (repeatable, combined with match-mode)")
	rootCmd.Flags().StringVarP(&regexPattern, "regex", "r", "", "regex pattern filter")
	rootCmd.Flags().StringSliceVarP(&levels, "level", "l", nil, "log level filter (DEBUG,INFO,WARN,ERROR,FATAL)")
	rootCmd.Flags().StringArrayVarP(&excludes, "exclude", "e", nil, "exclude lines containing pattern (repeatable)")
	rootCmd.Flags().StringVar(&matchMode, "match-mode", "or", "filter combination: 'and' or 'or'")

	// Context flags.
	rootCmd.Flags().IntVarP(&beforeLines, "before", "B", 0, "show N lines before each match")
	rootCmd.Flags().IntVarP(&afterLines, "after", "A", 0, "show N lines after each match")

	// I/O flags.
	rootCmd.Flags().StringVarP(&inputFile, "file", "f", "", "read from file instead of executing a command")
	rootCmd.Flags().BoolVar(&follow, "follow", false, "follow file for new lines (like tail -f)")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "write output to file")
	rootCmd.Flags().StringVar(&format, "format", "text", "output format: text, json")
	rootCmd.Flags().BoolVar(&color, "color", false, "colorize output by log level")

	// Stats and buffer flags.
	rootCmd.Flags().BoolVar(&showStats, "stats", false, "show summary statistics on exit")
	rootCmd.Flags().IntVar(&bufferSize, "buffer-size", 4096, "ring buffer capacity (entries)")
}

func run(cmd *cobra.Command, args []string) error {
	// Set up context with signal handling for graceful shutdown.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- Resolve source ---
	src, err := resolveSource(args)
	if err != nil {
		return err
	}

	// --- Build filter chain ---
	chain, err := buildFilterChain()
	if err != nil {
		return err
	}

	// Require at least one filter criterion.
	if chain.Len() == 0 {
		return fmt.Errorf("at least one filter flag is required: --keyword, --regex, or --level")
	}

	// --- Build context buffer ---
	var ctxBuf *filter.ContextBuffer
	if beforeLines > 0 || afterLines > 0 {
		ctxBuf = filter.NewContextBuffer(chain, beforeLines, afterLines)
	}

	// --- Build sinks ---
	sinks, err := buildSinks()
	if err != nil {
		return err
	}

	// --- Run pipeline ---
	stats := monitor.NewStats()
	cfg := &pipeline.Config{
		Source:    src,
		Filters:   chain,
		Sinks:     sinks,
		Context:   ctxBuf,
		Stats:     stats,
		RingBuf:   buffer.NewRing(bufferSize),
		ShowStats: showStats,
	}

	if err := pipeline.Run(ctx, cfg); err != nil {
		return err
	}

	return nil
}

// resolveSource determines the input source from flags and args.
func resolveSource(args []string) (source.Source, error) {
	// File source.
	if inputFile != "" {
		return source.NewFileSource(inputFile, follow), nil
	}

	// Stdin pipe (no args, data on stdin).
	if len(args) == 0 {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			return source.NewStdinSource(), nil
		}
		return nil, fmt.Errorf("no command provided and stdin is not a pipe\nUsage: lx [flags] -- <command> [args...]\n   or: <command> | lx [flags]")
	}

	// Exec source.
	if len(args) < 1 {
		return nil, fmt.Errorf("command is required when not using --file or stdin pipe")
	}

	command := args[0]
	var cmdArgs []string
	if len(args) > 1 {
		cmdArgs = args[1:]
	}
	return source.NewExecSource(command, cmdArgs), nil
}

// buildFilterChain assembles the filter chain from CLI flags.
func buildFilterChain() (*filter.Chain, error) {
	mode := filter.MatchAny
	if strings.EqualFold(matchMode, "and") {
		mode = filter.MatchAll
	}

	chain := filter.NewChain(mode)

	// Keyword filters.
	for _, kw := range keywords {
		chain.Add(filter.NewKeywordFilter(kw))
	}

	// Regex filter.
	if regexPattern != "" {
		rf, err := filter.NewRegexFilter(regexPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex: %w", err)
		}
		chain.Add(rf)
	}

	// Level filter.
	if len(levels) > 0 {
		var parsedLevels []entry.Level
		for _, l := range levels {
			parsed := entry.ParseLevel(l)
			if parsed == entry.LevelUnknown {
				return nil, fmt.Errorf("unknown log level: %q (valid: DEBUG, INFO, WARN, ERROR, FATAL)", l)
			}
			parsedLevels = append(parsedLevels, parsed)
		}
		chain.Add(filter.NewLevelFilter(parsedLevels...))
	}

	// Exclude filter (always AND, acts as a second-pass filter).
	// Exclude is applied separately in the pipeline if needed.
	if len(excludes) > 0 {
		chain.Add(filter.NewExcludeFilter(excludes...))
	}

	return chain, nil
}

// buildSinks assembles output sinks from CLI flags.
func buildSinks() ([]sink.Sink, error) {
	var sinks []sink.Sink

	// Primary output sink.
	switch format {
	case "json":
		sinks = append(sinks, sink.NewJSONSink(os.Stdout))
	default:
		sinks = append(sinks, sink.NewTerminalSink(os.Stdout, color))
	}

	// Optional file sink.
	if outputFile != "" {
		fs, err := sink.NewFileSink(outputFile, format)
		if err != nil {
			return nil, err
		}
		sinks = append(sinks, fs)
	}

	return sinks, nil
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
