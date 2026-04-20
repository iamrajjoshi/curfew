package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rajjoshi/curfew/internal/store"
)

var rangePresets = []int{7, 30, 90}

func (t *historyTab) clamp(history []store.HistoryRecord) {
	maxOffset := maxInt(0, len(history)-1)
	if t.scroll > maxOffset {
		t.scroll = maxOffset
	}
	if t.scroll < 0 {
		t.scroll = 0
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
	switch keyMsg.String() {
	case "up", "k":
		if t.scroll > 0 {
			t.scroll--
		}
	case "down", "j":
		if t.scroll < len(m.runtime.History)-1 {
			t.scroll++
		}
	case "1":
		m.historyDays = rangePresets[0]
		m.loading = true
		m.flash = "Refreshing history..."
		m.historyTab = t
		return m, m.loadSnapshot(false)
	case "2":
		m.historyDays = rangePresets[1]
		m.loading = true
		m.flash = "Refreshing history..."
		m.historyTab = t
		return m, m.loadSnapshot(false)
	case "3":
		m.historyDays = rangePresets[2]
		m.loading = true
		m.flash = "Refreshing history..."
		m.historyTab = t
		return m, m.loadSnapshot(false)
	}
	m.historyTab = t
	return m, nil
}

func (t historyTab) view(m model) string {
	lines := []string{"History", "-------"}
	lines = append(lines, fmt.Sprintf("Range: last %d days", m.historyDays))
	lines = append(lines, "Keys: [1] 7d  [2] 30d  [3] 90d  [up/down] scroll")
	lines = append(lines, "")
	if len(m.runtime.History) == 0 {
		lines = append(lines, "No history yet.")
		return strings.Join(lines, "\n")
	}

	for _, record := range visibleHistory(m.runtime.History, t.scroll, 12) {
		lines = append(lines, fmt.Sprintf("%s  %-9s snoozes=%d overrides=%d blocked=%d", record.Date, record.Status, record.SnoozesUsed, record.Overrides, record.BlockedCount))
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
	lines = append(lines, fmt.Sprintf("Respected nights: %d", m.runtime.Stats.RespectedNights))
	lines = append(lines, fmt.Sprintf("Snoozed nights: %d", m.runtime.Stats.SnoozedNights))
	lines = append(lines, fmt.Sprintf("Overridden nights: %d", m.runtime.Stats.OverriddenNights))
	lines = append(lines, fmt.Sprintf("Current streak: %d", m.runtime.Stats.CurrentStreak))
	lines = append(lines, fmt.Sprintf("Longest streak: %d", m.runtime.Stats.LongestStreak))
	if m.runtime.Stats.MostAttemptedCommand != "" {
		lines = append(lines, fmt.Sprintf("Most-attempted after-hours command: %s (%d)", m.runtime.Stats.MostAttemptedCommand, m.runtime.Stats.MostAttemptedCount))
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

func renderError(err error) string {
	return errorStyle.Render(fmt.Sprintf("curfew\n\n%s\n\nPress q to quit.", err))
}
