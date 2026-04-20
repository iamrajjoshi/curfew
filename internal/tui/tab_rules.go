package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/rajjoshi/curfew/internal/config"
	"github.com/rajjoshi/curfew/internal/rules"
)

func (t *rulesTab) clamp(cfg config.Config) {
	if t.selected < 0 {
		t.selected = 0
	}
	if len(cfg.Rules.Rule) == 0 {
		t.selected = 0
		return
	}
	if t.selected >= len(cfg.Rules.Rule) {
		t.selected = len(cfg.Rules.Rule) - 1
	}
}

func (t rulesTab) update(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && t.probeFocused {
		switch keyMsg.String() {
		case "esc", "enter":
			t.probeFocused = false
			t.probeInput.Blur()
			m.rulesTab = t
			return m, nil
		}
		var cmd tea.Cmd
		t.probeInput, cmd = t.probeInput.Update(msg)
		m.rulesTab = t
		return m, cmd
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "/":
		t.probeFocused = true
		cmd := t.probeInput.Focus()
		m.rulesTab = t
		return m, cmd
	case "up", "k":
		if t.selected > 0 {
			t.selected--
		}
	case "down", "j":
		if t.selected < len(m.draft.Rules.Rule)-1 {
			t.selected++
		}
	case "a":
		m.openRuleModal(-1)
	case "e", "enter":
		if len(m.draft.Rules.Rule) > 0 {
			m.openRuleModal(t.selected)
		}
	case "d":
		if len(m.draft.Rules.Rule) > 0 {
			m.openRuleDeleteModal(t.selected)
		}
	case "g":
		m.openRuleSettingsModal()
	case "J":
		if t.selected < len(m.draft.Rules.Rule)-1 {
			m.draft.Rules.Rule[t.selected], m.draft.Rules.Rule[t.selected+1] = m.draft.Rules.Rule[t.selected+1], m.draft.Rules.Rule[t.selected]
			t.selected++
			m.syncDraftState()
			m.flash = "Moved rule down in the draft config."
		}
	case "K":
		if t.selected > 0 && len(m.draft.Rules.Rule) > 0 {
			m.draft.Rules.Rule[t.selected], m.draft.Rules.Rule[t.selected-1] = m.draft.Rules.Rule[t.selected-1], m.draft.Rules.Rule[t.selected]
			t.selected--
			m.syncDraftState()
			m.flash = "Moved rule up in the draft config."
		}
	}

	m.rulesTab = t
	return m, nil
}

func (t rulesTab) view(m model) string {
	lines := []string{"Rules", "-----"}
	lines = append(lines, fmt.Sprintf("Default action: %s", m.draft.Rules.DefaultAction))
	lines = append(lines, fmt.Sprintf("Always allowed: %s", strings.Join(m.draft.Allowlist.Always, ", ")))
	lines = append(lines, "")
	lines = append(lines, "Try a command:")
	lines = append(lines, fmt.Sprintf("  %s", t.probeInput.View()))

	probe := strings.TrimSpace(t.probeInput.Value())
	if probe != "" {
		match := rules.Evaluate(m.draft, probe)
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
	if len(m.draft.Rules.Rule) == 0 {
		lines = append(lines, "  none")
	} else {
		for i, rule := range m.draft.Rules.Rule {
			cursor := " "
			if i == t.selected {
				cursor = ">"
			}
			lines = append(lines, fmt.Sprintf("%s %-6s %s", cursor, rule.Action, rule.Pattern))
		}
	}
	lines = append(lines, "")
	lines = append(lines, "Keys: [/] probe  [a] add  [e] edit  [d] delete  [g] defaults/allowlist  [J/K] reorder")
	return strings.Join(lines, "\n")
}

func (m *model) openRuleModal(index int) {
	pattern := ""
	action := "block"
	if index >= 0 && index < len(m.draft.Rules.Rule) {
		pattern = m.draft.Rules.Rule[index].Pattern
		action = m.draft.Rules.Rule[index].Action
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Pattern").Value(&pattern),
			huh.NewSelect[string]().
				Title("Action").
				Options(
					huh.NewOption("Allow", "allow"),
					huh.NewOption("Block", "block"),
					huh.NewOption("Warn", "warn"),
					huh.NewOption("Delay", "delay"),
				).
				Value(&action),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	title := "Add Rule"
	if index >= 0 {
		title = "Edit Rule"
	}
	modal := &modalState{
		title: title,
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			entry := config.RuleEntry{
				Pattern: strings.TrimSpace(pattern),
				Action:  strings.TrimSpace(strings.ToLower(action)),
			}
			if index >= 0 && index < len(next.draft.Rules.Rule) {
				next.draft.Rules.Rule[index] = entry
				next.flash = "Updated rule in the draft config."
			} else {
				next.draft.Rules.Rule = append(next.draft.Rules.Rule, entry)
				next.rulesTab.selected = len(next.draft.Rules.Rule) - 1
				next.flash = "Added rule to the draft config."
			}
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}

func (m *model) openRuleDeleteModal(index int) {
	if index < 0 || index >= len(m.draft.Rules.Rule) {
		return
	}
	target := m.draft.Rules.Rule[index]
	confirmed := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Delete rule %q?", target.Pattern)).
				Value(&confirmed),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	modal := &modalState{
		title: "Delete Rule",
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			if !confirmed {
				next.flash = "Delete cancelled."
				return true, nil
			}
			next.draft.Rules.Rule = append(next.draft.Rules.Rule[:index], next.draft.Rules.Rule[index+1:]...)
			next.rulesTab.clamp(next.draft)
			next.flash = "Removed rule from the draft config."
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}

func (m *model) openRuleSettingsModal() {
	defaultAction := m.draft.Rules.DefaultAction
	allowlist := strings.Join(m.draft.Allowlist.Always, ", ")
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Default action").
				Options(
					huh.NewOption("Allow", "allow"),
					huh.NewOption("Block", "block"),
					huh.NewOption("Warn", "warn"),
					huh.NewOption("Delay", "delay"),
				).
				Value(&defaultAction),
			huh.NewInput().
				Title("Always-allowed commands (comma separated)").
				Value(&allowlist),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	modal := &modalState{
		title: "Rule Settings",
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			next.draft.Rules.DefaultAction = strings.TrimSpace(strings.ToLower(defaultAction))
			next.draft.Allowlist.Always = splitCSV(allowlist)
			next.flash = "Updated default action and allowlist in the draft config."
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	output := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			output = append(output, trimmed)
		}
	}
	return output
}
