// Package tui provides an interactive terminal dashboard for real-time log monitoring.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Geun-Oh/lx/internal/buffer"
	"github.com/Geun-Oh/lx/internal/entry"
	"github.com/Geun-Oh/lx/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(1).
			PaddingRight(1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#353533"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAA00"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#44AAFF"))

	debugStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	highlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6600")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
)

// --- Messages ---

// LogMsg delivers a log entry to the TUI.
type LogMsg entry.LogEntry

// AlertMsg notifies the TUI that an alert was triggered.
type AlertMsg struct {
	Rules []string
	Entry entry.LogEntry
}

// SpikeMsg notifies the TUI that a rate spike was detected.
type SpikeMsg struct {
	Rate float64
}

// TickMsg triggers periodic UI updates.
type TickMsg time.Time

// DoneMsg signals the source has finished.
type DoneMsg struct{}

// --- Model ---

// Model is the bubbletea model for the TUI dashboard.
type Model struct {
	// Display state.
	logs       []string
	maxLines   int
	width      int
	height     int
	scrollPos  int // 0 = bottom (auto-scroll), >0 = scrolled up
	paused     bool
	pauseQueue []string

	// Search state.
	searching    bool
	searchQuery  string
	searchResult []int // indices into logs that match

	// Monitoring.
	Stats   *monitor.Stats
	Rate    *monitor.RateDetector
	Alerts  *monitor.AlertEngine
	RingBuf *buffer.Ring
	Source  string

	// Alert display.
	lastAlert  string
	alertFlash int // countdown for alert flash

	// Level counters.
	errorCount int
	warnCount  int
	totalCount int

	// Done state.
	done bool
}

// NewModel creates a new TUI model.
func NewModel(stats *monitor.Stats, rate *monitor.RateDetector, alerts *monitor.AlertEngine, ringBuf *buffer.Ring, sourceName string) Model {
	return Model{
		maxLines: 1000,
		Stats:    stats,
		Rate:     rate,
		Alerts:   alerts,
		RingBuf:  ringBuf,
		Source:   sourceName,
	}
}

// Init starts the tick timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.WindowSize())
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case LogMsg:
		return m.handleLog(msg)

	case AlertMsg:
		m.lastAlert = fmt.Sprintf("⚠ ALERT [%s]: %s", strings.Join(msg.Rules, ","), truncate(msg.Entry.Message, 60))
		m.alertFlash = 10
		return m, nil

	case SpikeMsg:
		m.lastAlert = fmt.Sprintf("📈 SPIKE: %.0f lines/s", msg.Rate)
		m.alertFlash = 8
		return m, nil

	case TickMsg:
		if m.alertFlash > 0 {
			m.alertFlash--
		}
		return m, tickCmd()

	case DoneMsg:
		m.done = true
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Search mode key handling.
	if m.searching {
		switch msg.String() {
		case "esc":
			m.searching = false
			m.searchQuery = ""
			m.searchResult = nil
			return m, nil
		case "enter":
			m.searching = false
			m.performSearch()
			return m, nil
		case "backspace":
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			}
			return m, nil
		default:
			if len(msg.String()) == 1 {
				m.searchQuery += msg.String()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "p":
		m.paused = !m.paused
		if !m.paused {
			m.logs = append(m.logs, m.pauseQueue...)
			m.pauseQueue = nil
			m.trimLogs()
		}
		return m, nil
	case "/":
		m.searching = true
		m.searchQuery = ""
		m.searchResult = nil
		return m, nil
	case "up", "k":
		if m.scrollPos < len(m.logs)-1 {
			m.scrollPos++
		}
		return m, nil
	case "down", "j":
		if m.scrollPos > 0 {
			m.scrollPos--
		}
		return m, nil
	case "g":
		m.scrollPos = 0 // jump to bottom (latest)
		return m, nil
	case "G":
		m.scrollPos = len(m.logs) - 1 // jump to top (oldest)
		return m, nil
	}

	return m, nil
}

func (m *Model) handleLog(msg LogMsg) (tea.Model, tea.Cmd) {
	e := entry.LogEntry(msg)
	line := m.formatLogLine(&e)

	m.totalCount++
	switch e.Level {
	case entry.LevelError, entry.LevelFatal:
		m.errorCount++
	case entry.LevelWarn:
		m.warnCount++
	}

	if m.paused {
		m.pauseQueue = append(m.pauseQueue, line)
		return m, nil
	}

	m.logs = append(m.logs, line)
	m.trimLogs()

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Title bar.
	title := titleStyle.Render(fmt.Sprintf(" lx monitor — %s ", m.Source))
	status := "▶ RUNNING"
	if m.paused {
		status = "⏸ PAUSED"
	}
	if m.done {
		status = "✔ DONE"
	}
	statusText := statusBarStyle.Render(fmt.Sprintf(" %s  %d lines ", status, m.totalCount))
	gap := m.width - lipgloss.Width(title) - lipgloss.Width(statusText)
	if gap < 0 {
		gap = 0
	}
	titleBar := title + statusBarStyle.Render(strings.Repeat(" ", gap)) + statusText
	sb.WriteString(titleBar)
	sb.WriteString("\n")

	// Alert bar (if active).
	if m.alertFlash > 0 && m.lastAlert != "" {
		alertBar := highlightStyle.Render(m.lastAlert)
		sb.WriteString(alertBar)
		sb.WriteString("\n")
	}

	// Search bar (if searching).
	if m.searching {
		searchBar := fmt.Sprintf(" 🔍 Search: %s█", m.searchQuery)
		sb.WriteString(searchBar)
		sb.WriteString("\n")
	}

	// Calculate viewport height.
	headerLines := 1 // title bar
	if m.alertFlash > 0 {
		headerLines++
	}
	if m.searching {
		headerLines++
	}
	footerLines := 2 // stats bar + help bar
	viewportHeight := m.height - headerLines - footerLines
	if viewportHeight < 1 {
		viewportHeight = 1
	}

	// Render log lines.
	visibleLogs := m.getVisibleLogs(viewportHeight)
	for _, line := range visibleLogs {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Pad remaining viewport.
	for i := len(visibleLogs); i < viewportHeight; i++ {
		sb.WriteString("\n")
	}

	// Stats bar.
	rate := m.Rate.CurrentRate()
	rateBar := m.renderRateBar(rate, 10)
	statsLine := fmt.Sprintf(" Rate: %s %.0f/s │ ERR: %d │ WARN: %d │ Total: %d",
		rateBar, rate, m.errorCount, m.warnCount, m.totalCount)
	if m.Alerts != nil && m.Alerts.TotalAlerts() > 0 {
		statsLine += fmt.Sprintf(" │ Alerts: %d", m.Alerts.TotalAlerts())
	}
	if m.scrollPos > 0 {
		statsLine += fmt.Sprintf(" │ ↑ %d", m.scrollPos)
	}
	sb.WriteString(statusBarStyle.Render(padRight(statsLine, m.width)))
	sb.WriteString("\n")

	// Help bar.
	helpText := " [/]Search  [p]Pause  [↑↓]Scroll  [g]Bottom  [q]Quit"
	if m.paused {
		helpText += fmt.Sprintf("  (queued: %d)", len(m.pauseQueue))
	}
	sb.WriteString(helpStyle.Render(helpText))

	return sb.String()
}

// --- Helpers ---

func (m *Model) formatLogLine(e *entry.LogEntry) string {
	ts := e.Timestamp.Format("15:04:05")
	levelStr := ""
	style := dimStyle

	switch e.Level {
	case entry.LevelError, entry.LevelFatal:
		style = errorStyle
		levelStr = e.Level.String() + " "
	case entry.LevelWarn:
		style = warnStyle
		levelStr = e.Level.String() + " "
	case entry.LevelInfo:
		style = infoStyle
		levelStr = e.Level.String() + " "
	case entry.LevelDebug:
		style = debugStyle
		levelStr = e.Level.String() + " "
	}

	msg := truncate(e.Message, m.width-25)
	return style.Render(fmt.Sprintf("%s [%s] %s%s", ts, e.Stream, levelStr, msg))
}

func (m *Model) getVisibleLogs(height int) []string {
	if len(m.logs) == 0 {
		return nil
	}

	end := len(m.logs) - m.scrollPos
	if end < 0 {
		end = 0
	}
	start := end - height
	if start < 0 {
		start = 0
	}

	// Highlight search results.
	result := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		line := m.logs[i]
		if m.searchQuery != "" && strings.Contains(line, m.searchQuery) {
			line = strings.ReplaceAll(line, m.searchQuery, highlightStyle.Render(m.searchQuery))
		}
		result = append(result, line)
	}
	return result
}

func (m *Model) performSearch() {
	m.searchResult = nil
	if m.searchQuery == "" {
		return
	}
	for i, line := range m.logs {
		if strings.Contains(line, m.searchQuery) {
			m.searchResult = append(m.searchResult, i)
		}
	}
	// Scroll to last match.
	if len(m.searchResult) > 0 {
		lastMatch := m.searchResult[len(m.searchResult)-1]
		m.scrollPos = len(m.logs) - lastMatch - 1
	}
}

func (m *Model) trimLogs() {
	if len(m.logs) > m.maxLines {
		excess := len(m.logs) - m.maxLines
		m.logs = m.logs[excess:]
	}
}

func (m *Model) renderRateBar(rate float64, width int) string {
	maxRate := 200.0 // scale: 200 lines/s = full bar
	filled := int(rate / maxRate * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
