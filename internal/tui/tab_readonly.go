package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iamrajjoshi/curfew/internal/store"
)

var rangePresets = []int{7, 30, 90}

const historyPageSize = 12

func (t *historyTab) clamp(history []store.HistoryRecord) {
	if len(history) == 0 {
		t.selected = 0
		t.scroll = 0
		return
	}

	maxIndex := len(history) - 1
	if t.selected > maxIndex {
		t.selected = maxIndex
	}
	if t.selected < 0 {
		t.selected = 0
	}

	maxScroll := maxInt(0, len(history)-historyPageSize)
	if t.scroll > maxScroll {
		t.scroll = maxScroll
	}
	if t.scroll < 0 {
		t.scroll = 0
	}

	if t.selected < t.scroll {
		t.scroll = t.selected
	}
	if t.selected >= t.scroll+historyPageSize {
		t.scroll = t.selected - historyPageSize + 1
	}
}

func (t *statsTab) clamp() {
	if t.scroll < 0 {
		t.scroll = 0
	}
}

func (t historyTab) update(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if t.detail != nil {
		switch keyMsg.String() {
		case "esc", "backspace":
			t.detail = nil
		}
		m.historyTab = t
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		t.selected--
		t.clamp(m.runtime.History)
	case "down", "j":
		t.selected++
		t.clamp(m.runtime.History)
	case "enter":
		if len(m.runtime.History) == 0 {
			m.historyTab = t
			return m, nil
		}
		record := m.runtime.History[t.selected]
		m.loading = true
		m.flash = fmt.Sprintf("Loading history for %s...", record.Date)
		m.historyTab = t
		return m, m.loadHistoryDetail(record.Date)
	case "1":
		m.historyDays = rangePresets[0]
		m.loading = true
		m.flash = "Refreshing history..."
		t.detail = nil
		m.historyTab = t
		return m, m.loadSnapshot(false)
	case "2":
		m.historyDays = rangePresets[1]
		m.loading = true
		m.flash = "Refreshing history..."
		t.detail = nil
		m.historyTab = t
		return m, m.loadSnapshot(false)
	case "3":
		m.historyDays = rangePresets[2]
		m.loading = true
		m.flash = "Refreshing history..."
		t.detail = nil
		m.historyTab = t
		return m, m.loadSnapshot(false)
	}
	m.historyTab = t
	return m, nil
}

func (t historyTab) view(m model) string {
	if t.detail != nil {
		return renderHistoryDetailView(*t.detail, m.historyDays)
	}

	lines := []string{"History", "-------"}
	lines = append(lines, fmt.Sprintf("Range: last %d days", m.historyDays))
	lines = append(lines, "Keys: [1] 7d  [2] 30d  [3] 90d  [up/down] move  [enter] details")
	lines = append(lines, "")
	if len(m.runtime.History) == 0 {
		lines = append(lines, "No history yet.")
		return strings.Join(lines, "\n")
	}

	for index, record := range visibleHistory(m.runtime.History, t.scroll, historyPageSize) {
		actualIndex := t.scroll + index
		prefix := "  "
		if actualIndex == t.selected {
			prefix = "> "
		}
		lines = append(lines, prefix+renderHistorySummary(record))
	}
	return strings.Join(lines, "\n")
}

func (t statsTab) update(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "1":
		m.statsDays = rangePresets[0]
		m.loading = true
		m.flash = "Refreshing stats..."
		m.statsTab = t
		return m, m.loadSnapshot(false)
	case "2":
		m.statsDays = rangePresets[1]
		m.loading = true
		m.flash = "Refreshing stats..."
		m.statsTab = t
		return m, m.loadSnapshot(false)
	case "3":
		m.statsDays = rangePresets[2]
		m.loading = true
		m.flash = "Refreshing stats..."
		m.statsTab = t
		return m, m.loadSnapshot(false)
	}
	return m, nil
}

func (t statsTab) view(m model) string {
	lines := []string{"Stats", "-----"}
	lines = append(lines, fmt.Sprintf("Range: last %d days", m.statsDays))
	lines = append(lines, "Keys: [1] 7d  [2] 30d  [3] 90d")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Total nights: %d", m.runtime.Stats.TotalNights))
	lines = append(lines, fmt.Sprintf("Respected nights: %d", m.runtime.Stats.RespectedNights))
	lines = append(lines, fmt.Sprintf("Snoozed nights: %d", m.runtime.Stats.SnoozedNights))
	lines = append(lines, fmt.Sprintf("Overridden nights: %d", m.runtime.Stats.OverriddenNights))
	lines = append(lines, fmt.Sprintf("Adherent nights: %d", m.runtime.Stats.AdherentNights))
	lines = append(lines, fmt.Sprintf("Adherence rate: %.1f%%", m.runtime.Stats.AdherenceRate*100))
	lines = append(lines, fmt.Sprintf("Current streak: %d", m.runtime.Stats.CurrentStreak))
	lines = append(lines, fmt.Sprintf("Longest streak: %d", m.runtime.Stats.LongestStreak))
	lines = append(lines, "")
	lines = append(lines, "Top after-hours commands:")
	if len(m.runtime.Stats.TopCommands) == 0 {
		lines = append(lines, "  none")
	} else {
		for _, command := range m.runtime.Stats.TopCommands {
			lines = append(lines, fmt.Sprintf("  %s (%d)", command.Command, command.Count))
		}
	}
	return strings.Join(lines, "\n")
}

func historySparkline(history []store.HistoryRecord, nights int) string {
	if len(history) == 0 || nights <= 0 {
		return "......."
	}
	segments := make([]string, 0, nights)
	for i := 0; i < nights; i++ {
		if i >= len(history) {
			segments = append(segments, ".")
			continue
		}
		switch history[i].Status {
		case "respected":
			segments = append(segments, "#")
		case "snoozed":
			segments = append(segments, "~")
		default:
			segments = append(segments, "x")
		}
	}
	return strings.Join(segments, "")
}

func visibleHistory(history []store.HistoryRecord, offset int, limit int) []store.HistoryRecord {
	if offset >= len(history) {
		return nil
	}
	end := offset + limit
	if end > len(history) {
		end = len(history)
	}
	return history[offset:end]
}

func renderHistorySummary(record store.HistoryRecord) string {
	lastCommand := "n/a"
	if record.LastCommand != nil {
		lastCommand = record.LastCommand.Format("2006-01-02 15:04 MST")
	}
	return fmt.Sprintf(
		"%s  %-9s snoozes=%d overrides=%d blocked=%d last=%s",
		record.Date,
		record.Status,
		record.SnoozesUsed,
		record.Overrides,
		record.BlockedCount,
		lastCommand,
	)
}

func renderHistoryDetailView(details store.SessionDetails, days int) string {
	lines := []string{"History Detail", "--------------"}
	lines = append(lines, fmt.Sprintf("Range: last %d days", days))
	lines = append(lines, "Keys: [esc/backspace] back")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Date: %s", details.Session.Date))
	lines = append(lines, fmt.Sprintf("Status: %s", details.Session.Status))
	lines = append(lines, fmt.Sprintf("Configured schedule: %s -> %s", details.Session.BedtimeConfigured, details.Session.WakeConfigured))
	lines = append(lines, fmt.Sprintf("Snoozes used: %d", details.Session.SnoozesUsed))
	lines = append(lines, fmt.Sprintf("Skipped: %t", details.Session.Skipped))
	lines = append(lines, fmt.Sprintf("Forced active: %t", details.Session.ForcedActive))
	lines = append(lines, fmt.Sprintf("Blocked attempts: %d", details.Session.BlockedCount))
	lines = append(lines, fmt.Sprintf("Overrides: %d", details.Session.Overrides))
	if details.Session.FirstBlockedAt != nil {
		lines = append(lines, fmt.Sprintf("First blocked at: %s", details.Session.FirstBlockedAt.Format("2006-01-02 15:04 MST")))
	}
	if details.Session.LastCommandAt != nil {
		lines = append(lines, fmt.Sprintf("Last command at: %s", details.Session.LastCommandAt.Format("2006-01-02 15:04 MST")))
	}
	lines = append(lines, "", "Events", "------")
	if len(details.Events) == 0 {
		lines = append(lines, "No intercepted commands.")
		return strings.Join(lines, "\n")
	}
	for _, event := range details.Events {
		line := fmt.Sprintf("%s  %s  action=%s outcome=%s", event.Timestamp.Format("2006-01-02 15:04 MST"), event.Command, event.Action, event.Outcome)
		if event.Tier != "" {
			line += fmt.Sprintf(" tier=%s", event.Tier)
		}
		if event.MatchedRule != "" {
			line += fmt.Sprintf(" rule=%q", event.MatchedRule)
		}
		if event.Shell != "" {
			line += fmt.Sprintf(" shell=%s", event.Shell)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderError(err error) string {
	return errorStyle.Render(fmt.Sprintf("curfew\n\n%s\n\nPress q to quit.", err))
}
