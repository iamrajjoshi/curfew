package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rajjoshi/curfew/internal/app"
	"github.com/rajjoshi/curfew/internal/config"
	"github.com/rajjoshi/curfew/internal/friction"
	"github.com/rajjoshi/curfew/internal/rules"
	"github.com/rajjoshi/curfew/internal/schedule"
	"github.com/rajjoshi/curfew/internal/store"
)

type tab struct {
	key   string
	label string
}

var tabs = []tab{
	{key: "dashboard", label: "Dashboard"},
	{key: "schedule", label: "Schedule"},
	{key: "rules", label: "Rules"},
	{key: "override", label: "Override"},
	{key: "history", label: "History"},
	{key: "stats", label: "Stats"},
}

type snapshot struct {
	Config  config.Config
	Status  app.Status
	Rules   []config.RuleEntry
	History []store.HistoryRecord
	Stats   store.Stats
}

type snapshotMsg struct {
	data snapshot
	err  error
}

type snoozeMsg struct {
	status app.Status
	err    error
}

type model struct {
	app       *app.App
	activeTab int
	width     int
	height    int
	helpOpen  bool
	loading   bool
	flash     string
	err       error
	snapshot  snapshot
	ruleInput textinput.Model
}

func Run(application *app.App, initialTab string) error {
	input := textinput.New()
	input.Placeholder = "try a command"
	input.CharLimit = 200
	input.Width = 50
	input.Focus()

	m := model{
		app:       application,
		activeTab: tabIndex(initialTab),
		loading:   true,
		ruleInput: input,
	}

	program := tea.NewProgram(m, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return m.loadSnapshot()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		return m, nil
	case tea.KeyMsg:
		if m.loading {
			if message.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		switch message.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "l", "right":
			m.activeTab = (m.activeTab + 1) % len(tabs)
			return m, nil
		case "shift+tab", "h", "left":
			m.activeTab = (m.activeTab + len(tabs) - 1) % len(tabs)
			return m, nil
		case "?":
			m.helpOpen = !m.helpOpen
			return m, nil
		case "r":
			m.loading = true
			m.flash = "Refreshing..."
			return m, m.loadSnapshot()
		case "s":
			if tabs[m.activeTab].key == "dashboard" {
				return m, m.snooze()
			}
		}

		if tabs[m.activeTab].key == "rules" {
			var cmd tea.Cmd
			m.ruleInput, cmd = m.ruleInput.Update(message)
			return m, cmd
		}
	case snapshotMsg:
		m.loading = false
		m.err = message.err
		if message.err == nil {
			m.snapshot = message.data
			if m.flash == "Refreshing..." {
				m.flash = "Refreshed."
			}
		}
		return m, nil
	case snoozeMsg:
		if message.err != nil {
			m.flash = message.err.Error()
		} else {
			m.snapshot.Status = message.status
			m.flash = fmt.Sprintf("Snoozed until %s.", message.status.SnoozedUntil)
		}
		return m, nil
	}

	if tabs[m.activeTab].key == "rules" {
		var cmd tea.Cmd
		m.ruleInput, cmd = m.ruleInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.loading {
		return "Loading curfew..."
	}
	if m.err != nil {
		return renderError(m.err)
	}

	var sections []string
	sections = append(sections, renderTabs(m.activeTab))
	sections = append(sections, renderBody(m))
	if m.flash != "" {
		sections = append(sections, flashStyle.Render(m.flash))
	}
	if m.helpOpen {
		sections = append(sections, helpStyle.Render("tab/shift-tab switch tabs, r refresh, q quit, ? toggle help, s snooze on Dashboard"))
	} else {
		sections = append(sections, mutedStyle.Render("Press ? for help"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m model) loadSnapshot() tea.Cmd {
	return func() tea.Msg {
		cfg, err := m.app.LoadConfig()
		if err != nil {
			return snapshotMsg{err: err}
		}
		status, err := m.app.EvaluateStatus()
		if err != nil {
			return snapshotMsg{err: err}
		}
		ruleList, err := m.app.Rules()
		if err != nil {
			return snapshotMsg{err: err}
		}
		history, err := m.app.History(context.Background(), 30)
		if err != nil {
			return snapshotMsg{err: err}
		}
		stats, err := m.app.Stats(context.Background(), 30)
		if err != nil {
			return snapshotMsg{err: err}
		}

		return snapshotMsg{
			data: snapshot{
				Config:  cfg,
				Status:  status,
				Rules:   ruleList,
				History: history,
				Stats:   stats,
			},
		}
	}
}

func (m model) snooze() tea.Cmd {
	return func() tea.Msg {
		duration, err := time.ParseDuration(m.snapshot.Config.Grace.SnoozeDuration)
		if err != nil {
			return snoozeMsg{err: err}
		}
		_, _, _, err = m.app.Snooze(context.Background(), duration)
		if err != nil {
			return snoozeMsg{err: err}
		}
		status, err := m.app.EvaluateStatus()
		return snoozeMsg{status: status, err: err}
	}
}

func renderBody(m model) string {
	body := baseBodyStyle
	if m.width > 0 {
		body = body.Width(maxInt(40, m.width-4))
	}

	switch tabs[m.activeTab].key {
	case "dashboard":
		return body.Render(renderDashboard(m.snapshot.Status, m.snapshot.Rules, m.snapshot.History))
	case "schedule":
		return body.Render(renderSchedule(m.snapshot.Config))
	case "rules":
		return body.Render(renderRules(m.snapshot.Config, m.snapshot.Rules, m.ruleInput.Value()))
	case "override":
		return body.Render(renderOverride(m.snapshot.Config))
	case "history":
		return body.Render(renderHistory(m.snapshot.History))
	case "stats":
		return body.Render(renderStats(m.snapshot.Stats))
	default:
		return body.Render("Unknown tab.")
	}
}

func renderTabs(active int) string {
	rendered := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		if i == active {
			rendered = append(rendered, activeTabStyle.Render(tab.label))
			continue
		}
		rendered = append(rendered, inactiveTabStyle.Render(tab.label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func renderDashboard(status app.Status, rules []config.RuleEntry, history []store.HistoryRecord) string {
	lines := []string{"Status"}
	lines = append(lines, "------")
	lines = append(lines, status.Render())
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Rules active: %d", len(rules)))
	lines = append(lines, fmt.Sprintf("Last 7 nights: %s", historySparkline(history, 7)))
	lines = append(lines, "")
	lines = append(lines, "Shortcuts: [s] snooze, [r] refresh")
	return strings.Join(lines, "\n")
}

func renderSchedule(cfg config.Config) string {
	lines := []string{"Schedule", "--------"}
	lines = append(lines, fmt.Sprintf("Timezone: %s", cfg.Schedule.Timezone))
	lines = append(lines, fmt.Sprintf("Default: bedtime %s -> wake %s", cfg.Schedule.Default.Bedtime, cfg.Schedule.Default.Wake))
	lines = append(lines, "")
	lines = append(lines, "Overrides:")
	if len(cfg.Schedule.Overrides) == 0 {
		lines = append(lines, "  none")
	} else {
		for _, day := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
			if override, ok := cfg.Schedule.Overrides[day]; ok {
				lines = append(lines, fmt.Sprintf("  %-9s bedtime %s -> wake %s", day, override.Bedtime, override.Wake))
			}
		}
	}
	lines = append(lines, "")
	lines = append(lines, "Grace")
	lines = append(lines, fmt.Sprintf("  warning window: %s", cfg.Grace.WarningWindow))
	lines = append(lines, fmt.Sprintf("  hard stop: %s", cfg.Grace.HardStopAfter))
	lines = append(lines, fmt.Sprintf("  snoozes per night: %d", cfg.Grace.SnoozeMaxPerNight))
	lines = append(lines, fmt.Sprintf("  snooze duration: %s", cfg.Grace.SnoozeDuration))
	return strings.Join(lines, "\n")
}

func renderRules(cfg config.Config, rulesList []config.RuleEntry, probe string) string {
	lines := []string{"Rules", "-----"}
	lines = append(lines, fmt.Sprintf("Default action: %s", cfg.Rules.DefaultAction))
	lines = append(lines, fmt.Sprintf("Always allowed: %s", strings.Join(cfg.Allowlist.Always, ", ")))
	lines = append(lines, "")
	lines = append(lines, "Try a command:")
	lines = append(lines, fmt.Sprintf("  %s", probe))

	probe = strings.TrimSpace(probe)
	if probe != "" {
		match := rules.Evaluate(cfg, probe)
		lines = append(lines, fmt.Sprintf("Matched action: %s", match.Action))
		if match.AllowedByAllowlist {
			lines = append(lines, fmt.Sprintf("Matched via allowlist on %q", match.CommandWord))
		} else if match.Matched {
			lines = append(lines, fmt.Sprintf("Matched rule %q (%s)", match.Pattern, match.Kind))
		} else {
			lines = append(lines, "No explicit rule matched.")
		}
		lines = append(lines, "")
	}

	lines = append(lines, "Rules:")
	for _, rule := range rulesList {
		lines = append(lines, fmt.Sprintf("  %-6s %s", rule.Action, rule.Pattern))
	}
	return strings.Join(lines, "\n")
}

func renderOverride(cfg config.Config) string {
	lines := []string{"Override", "--------"}
	lines = append(lines, fmt.Sprintf("Preset: %s", cfg.Override.Preset))
	lines = append(lines, fmt.Sprintf("Passphrase: %s", cfg.Override.Passphrase))
	lines = append(lines, "")
	lines = append(lines, "Blocked-command preview by tier:")
	for _, tier := range []schedule.Tier{schedule.TierWarning, schedule.TierCurfew, schedule.TierHardStop} {
		profile := friction.EffectiveProfile(cfg, "block", tier)
		lines = append(lines, fmt.Sprintf("  %-9s %s", tier, describeProfile(profile)))
	}
	lines = append(lines, "")
	lines = append(lines, "Warn-rule preview by tier:")
	for _, tier := range []schedule.Tier{schedule.TierWarning, schedule.TierCurfew, schedule.TierHardStop} {
		profile := friction.EffectiveProfile(cfg, "warn", tier)
		lines = append(lines, fmt.Sprintf("  %-9s %s", tier, describeProfile(profile)))
	}
	return strings.Join(lines, "\n")
}

func renderHistory(history []store.HistoryRecord) string {
	lines := []string{"History", "-------"}
	if len(history) == 0 {
		lines = append(lines, "No history yet.")
		return strings.Join(lines, "\n")
	}
	for _, record := range history {
		lines = append(lines, fmt.Sprintf("%s  %-9s snoozes=%d overrides=%d blocked=%d", record.Date, record.Status, record.SnoozesUsed, record.Overrides, record.BlockedCount))
	}
	return strings.Join(lines, "\n")
}

func renderStats(stats store.Stats) string {
	lines := []string{"Stats", "-----"}
	lines = append(lines, fmt.Sprintf("Respected nights: %d", stats.RespectedNights))
	lines = append(lines, fmt.Sprintf("Snoozed nights: %d", stats.SnoozedNights))
	lines = append(lines, fmt.Sprintf("Overridden nights: %d", stats.OverriddenNights))
	lines = append(lines, fmt.Sprintf("Current streak: %d", stats.CurrentStreak))
	lines = append(lines, fmt.Sprintf("Longest streak: %d", stats.LongestStreak))
	if stats.MostAttemptedCommand != "" {
		lines = append(lines, fmt.Sprintf("Most-attempted after-hours command: %s (%d)", stats.MostAttemptedCommand, stats.MostAttemptedCount))
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

func describeProfile(profile friction.Profile) string {
	switch profile.Kind {
	case friction.KindAllow:
		return "allow"
	case friction.KindBanner:
		return "banner only"
	case friction.KindPrompt:
		return "prompt"
	case friction.KindWait:
		return fmt.Sprintf("wait %ds", profile.WaitSeconds)
	case friction.KindPassphrase:
		return "passphrase"
	case friction.KindMath:
		return fmt.Sprintf("math (%s)", profile.MathDifficulty)
	case friction.KindCombined:
		if profile.Passphrase != "" {
			return fmt.Sprintf("wait %ds + passphrase", profile.WaitSeconds)
		}
		return fmt.Sprintf("wait %ds + math", profile.WaitSeconds)
	case friction.KindBlock:
		return "block"
	default:
		return string(profile.Kind)
	}
}

func renderError(err error) string {
	return errorStyle.Render(fmt.Sprintf("curfew\n\n%s\n\nPress q to quit.", err))
}

func tabIndex(key string) int {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return 0
	}
	for i, tab := range tabs {
		if tab.key == key {
			return i
		}
	}
	return 0
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

var (
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")).
			Padding(0, 1)
	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Background(lipgloss.Color("238")).
				Padding(0, 1)
	baseBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("24")).
			Padding(1, 2)
	flashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("194"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Bold(true)
)
