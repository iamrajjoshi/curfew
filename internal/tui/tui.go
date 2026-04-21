package tui

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/iamrajjoshi/curfew/internal/app"
	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/store"
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

type runtimeSnapshot struct {
	Status  app.Status
	History []store.HistoryRecord
	Stats   store.Stats
}

type snapshotMsg struct {
	config    *config.Config
	runtime   runtimeSnapshot
	syncDraft bool
	err       error
}

type saveMsg struct {
	err error
}

type actionMsg struct {
	flash  string
	err    error
	reload bool
}

type modalSubmittedMsg struct{}
type modalCancelledMsg struct{}

type modalState struct {
	title string
	form  *huh.Form
	apply func(*model) (bool, tea.Cmd)
}

type dashboardTab struct{}

type scheduleTab struct {
	selectedDay int
}

type rulesTab struct {
	selected     int
	probeInput   textinput.Model
	probeFocused bool
}

type overrideTab struct{}

type historyTab struct {
	scroll int
}

type statsTab struct {
	scroll int
}

type model struct {
	app          *app.App
	activeTab    int
	width        int
	height       int
	helpOpen     bool
	loading      bool
	flash        string
	err          error
	confirmQuit  bool
	historyDays  int
	statsDays    int
	persisted    config.Config
	draft        config.Config
	dirty        bool
	validation   error
	runtime      runtimeSnapshot
	modal        *modalState
	dashboardTab dashboardTab
	scheduleTab  scheduleTab
	rulesTab     rulesTab
	overrideTab  overrideTab
	historyTab   historyTab
	statsTab     statsTab
}

func Run(application *app.App, initialTab string) error {
	program := tea.NewProgram(newModel(application, initialTab), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newModel(application *app.App, initialTab string) model {
	probe := textinput.New()
	probe.Placeholder = "try a command"
	probe.CharLimit = 200
	probe.Width = 50

	return model{
		app:         application,
		activeTab:   tabIndex(initialTab),
		loading:     true,
		historyDays: 30,
		statsDays:   30,
		scheduleTab: scheduleTab{selectedDay: 0},
		rulesTab: rulesTab{
			probeInput: probe,
		},
	}
}

func (m model) Init() tea.Cmd {
	return m.loadSnapshot(true)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		if m.modal != nil {
			m.modal.form.WithWidth(maxInt(40, m.width-12))
		}
		return m, nil
	case snapshotMsg:
		m.loading = false
		if message.err != nil {
			m.err = message.err
			return m, nil
		}
		m.err = nil
		wasDirty := m.dirty
		previousPersisted := m.persisted
		if message.config != nil {
			m.persisted = config.Clone(*message.config)
			if message.syncDraft {
				m.draft = config.Clone(*message.config)
				m.confirmQuit = false
			}
			m.syncDraftState()
		}
		m.runtime = message.runtime
		m.rulesTab.clamp(m.draft)
		m.historyTab.clamp(m.runtime.History)
		m.statsTab.clamp()
		configChanged := !reflect.DeepEqual(previousPersisted, m.persisted)
		if !message.syncDraft && configChanged && !wasDirty && m.dirty {
			m.flash = "Config changed on disk. Press ctrl+r to reload it or ctrl+s to overwrite."
		} else if m.flash == "Refreshing..." {
			m.flash = "Refreshed."
		}
		return m, nil
	case saveMsg:
		m.loading = false
		if message.err != nil {
			m.flash = message.err.Error()
			return m, nil
		}
		m.flash = "Saved config."
		m.loading = true
		return m, m.loadSnapshot(true)
	case actionMsg:
		if message.err != nil {
			m.flash = message.err.Error()
			m.loading = false
			return m, nil
		}
		if message.flash != "" {
			m.flash = message.flash
		}
		if message.reload {
			m.loading = true
			return m, m.loadSnapshot(false)
		}
		return m, nil
	case disablePromptMsg:
		m.flash = "Override still requested."
		m.openDisableModal(message.title, message.request, message.skipped, message.challenge)
		return m, nil
	case modalSubmittedMsg:
		if m.modal == nil {
			return m, nil
		}
		closeModal, cmd := m.modal.apply(&m)
		if closeModal {
			m.modal = nil
			m.syncDraftState()
		}
		return m, cmd
	case modalCancelledMsg:
		m.modal = nil
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.modal != nil {
			updated, cmd := m.modal.form.Update(keyMsg)
			m.modal.form = updated.(*huh.Form)
			return m, cmd
		}
		if tabs[m.activeTab].key == "rules" && m.rulesTab.probeFocused {
			switch keyMsg.String() {
			case "ctrl+c", "ctrl+s", "ctrl+r":
			default:
				return m.dispatchTabUpdate(msg)
			}
		}

		switch keyMsg.String() {
		case "ctrl+c":
			if m.dirty && !m.confirmQuit {
				m.confirmQuit = true
				m.flash = "Unsaved changes. Press q or ctrl+c again to quit without saving."
				return m, nil
			}
			return m, tea.Quit
		case "q":
			if m.dirty && !m.confirmQuit {
				m.confirmQuit = true
				m.flash = "Unsaved changes. Press q again to quit without saving, or ctrl+s to save."
				return m, nil
			}
			return m, tea.Quit
		case "ctrl+s":
			if m.validation != nil {
				m.flash = fmt.Sprintf("Fix validation errors before saving: %v", m.validation)
				return m, nil
			}
			m.loading = true
			m.flash = "Saving..."
			return m, m.saveDraft()
		case "ctrl+r":
			if !m.dirty {
				m.flash = "No unsaved changes to discard."
				return m, nil
			}
			m.loading = true
			m.flash = "Reloading config from disk..."
			return m, m.loadSnapshot(true)
		case "?":
			m.helpOpen = !m.helpOpen
			return m, nil
		case "tab", "l", "right":
			m.activeTab = (m.activeTab + 1) % len(tabs)
			m.confirmQuit = false
			return m, nil
		case "shift+tab", "h", "left":
			m.activeTab = (m.activeTab + len(tabs) - 1) % len(tabs)
			m.confirmQuit = false
			return m, nil
		case "r":
			m.loading = true
			m.flash = "Refreshing..."
			return m, m.loadSnapshot(false)
		}
	}

	return m.dispatchTabUpdate(msg)
}

func (m model) View() string {
	if m.loading && reflect.DeepEqual(m.persisted, config.Config{}) {
		return "Loading curfew..."
	}
	if m.err != nil {
		return renderError(m.err)
	}

	body := m.renderBody()
	sections := []string{
		renderTabs(m.activeTab, m.dirty),
		body,
	}

	if m.validation != nil && isEditableTab(tabs[m.activeTab].key) {
		sections = append(sections, errorStyle.Render(fmt.Sprintf("Draft validation: %v", m.validation)))
	}
	if m.flash != "" {
		sections = append(sections, flashStyle.Render(m.flash))
	}
	if m.helpOpen {
		sections = append(sections, helpStyle.Render(m.helpText()))
	} else {
		sections = append(sections, mutedStyle.Render("Press ? for help"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	if m.modal == nil {
		return content
	}

	modal := modalStyle.Width(maxInt(50, m.width-10)).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render(m.modal.title),
			"",
			m.modal.form.View(),
		),
	)
	return lipgloss.JoinVertical(lipgloss.Left, content, "", modal)
}

func (m model) dispatchTabUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch tabs[m.activeTab].key {
	case "dashboard":
		return m.dashboardTab.update(msg, m)
	case "schedule":
		return m.scheduleTab.update(msg, m)
	case "rules":
		return m.rulesTab.update(msg, m)
	case "override":
		return m.overrideTab.update(msg, m)
	case "history":
		return m.historyTab.update(msg, m)
	case "stats":
		return m.statsTab.update(msg, m)
	default:
		return m, nil
	}
}

func (m model) renderBody() string {
	body := baseBodyStyle
	if m.width > 0 {
		body = body.Width(maxInt(40, m.width-4))
	}

	var content string
	switch tabs[m.activeTab].key {
	case "dashboard":
		content = m.dashboardTab.view(m)
	case "schedule":
		content = m.scheduleTab.view(m)
	case "rules":
		content = m.rulesTab.view(m)
	case "override":
		content = m.overrideTab.view(m)
	case "history":
		content = m.historyTab.view(m)
	case "stats":
		content = m.statsTab.view(m)
	default:
		content = "Unknown tab."
	}
	return body.Render(content)
}

func (m *model) loadSnapshot(syncDraft bool) tea.Cmd {
	historyDays := m.historyDays
	statsDays := m.statsDays
	return func() tea.Msg {
		cfg, err := m.app.LoadConfig()
		if err != nil {
			return snapshotMsg{err: err}
		}
		status, err := m.app.EvaluateStatus()
		if err != nil {
			return snapshotMsg{err: err}
		}
		history, err := m.app.History(context.Background(), historyDays)
		if err != nil {
			return snapshotMsg{err: err}
		}
		stats, err := m.app.Stats(context.Background(), statsDays)
		if err != nil {
			return snapshotMsg{err: err}
		}
		return snapshotMsg{
			config:    &cfg,
			runtime:   runtimeSnapshot{Status: status, History: history, Stats: stats},
			syncDraft: syncDraft,
		}
	}
}

func (m *model) saveDraft() tea.Cmd {
	draft := config.Clone(m.draft)
	return func() tea.Msg {
		return saveMsg{err: m.app.SaveConfig(draft)}
	}
}

func (m *model) syncDraftState() {
	m.validation = m.draft.Validate()
	m.dirty = !reflect.DeepEqual(m.draft, m.persisted)
	m.rulesTab.clamp(m.draft)
	if !m.dirty {
		m.confirmQuit = false
	}
}

func (m model) helpText() string {
	line := "tab/shift-tab switch tabs, ctrl+s save, ctrl+r discard draft, r refresh runtime data, q quit"
	switch tabs[m.activeTab].key {
	case "dashboard":
		return line + ", s snooze, f force-enable, x stop tonight, k skip tonight"
	case "schedule":
		return line + ", g edit general schedule, e edit selected day override, d clear override"
	case "rules":
		return line + ", / edit probe, a add rule, e edit, d delete, g edit defaults, J/K reorder"
	case "override":
		return line + ", p edit preset settings, 1/2/3 edit warning/curfew/hard-stop custom tiers"
	case "history", "stats":
		return line + ", 1/2/3 set range to 7/30/90 days, up/down scroll"
	default:
		return line
	}
}

func renderTabs(active int, dirty bool) string {
	rendered := make([]string, 0, len(tabs))
	for i, tab := range tabs {
		label := tab.label
		if dirty && i == active {
			label += " *"
		}
		if i == active {
			rendered = append(rendered, activeTabStyle.Render(label))
			continue
		}
		rendered = append(rendered, inactiveTabStyle.Render(label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
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

func isEditableTab(key string) bool {
	switch key {
	case "schedule", "rules", "override":
		return true
	default:
		return false
	}
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
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("69")).
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
