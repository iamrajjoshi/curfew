package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/rajjoshi/curfew/internal/friction"
	"github.com/rajjoshi/curfew/internal/schedule"
)

func (t dashboardTab) update(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "s":
		return m, m.snoozeCmd()
	case "f":
		return m, m.forceEnableCmd()
	case "x":
		return m, m.beginDisableModal(false)
	case "k":
		return m, m.beginDisableModal(true)
	default:
		return m, nil
	}
}

func (t dashboardTab) view(m model) string {
	lines := []string{"Status", "------"}
	lines = append(lines, m.runtime.Status.Render())
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Rules active: %d", len(m.draft.Rules.Rule)))
	lines = append(lines, fmt.Sprintf("Last 7 nights: %s", historySparkline(m.runtime.History, 7)))
	if m.dirty {
		lines = append(lines, "Draft config changes are pending save.")
	}
	lines = append(lines, "")
	lines = append(lines, "Actions: [s] snooze  [f] force-enable  [x] stop tonight  [k] skip tonight  [r] refresh")
	return strings.Join(lines, "\n")
}

func (m model) snoozeCmd() tea.Cmd {
	duration, err := time.ParseDuration(m.persisted.Grace.SnoozeDuration)
	if err != nil {
		return func() tea.Msg { return actionMsg{err: err} }
	}
	return func() tea.Msg {
		_, _, _, err := m.app.Snooze(context.Background(), duration)
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{flash: "Curfew snoozed.", reload: true}
	}
}

func (m model) forceEnableCmd() tea.Cmd {
	return func() tea.Msg {
		session, err := m.app.SetForcedActive(context.Background())
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			flash:  fmt.Sprintf("Curfew force-enabled for session %s.", session.Date),
			reload: true,
		}
	}
}

func (m *model) beginDisableModal(skipped bool) tea.Cmd {
	reason := "Disable curfew for the rest of this session?"
	if skipped {
		reason = "Skip tonight's curfew?"
	}
	request, err := m.app.DisableRequest(skipped, reason)
	if err != nil {
		m.flash = err.Error()
		return nil
	}
	if request.Tier == schedule.TierHardStop || request.Profile.Kind == friction.KindBlock {
		m.flash = "Curfew cannot be disabled during hard stop."
		return nil
	}

	challenge := friction.NewChallenge(request.Profile, request.Reason)
	answer := ""
	form := huhFormForChallenge(challenge, &answer)
	m.modal = &modalState{
		title: "Stop Tonight",
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			allowed, _ := challenge.Check(answer)
			if !allowed {
				next.flash = "Override cancelled."
				return true, nil
			}
			if challenge.WaitSeconds > 0 {
				next.flash = fmt.Sprintf("Waiting %ds before applying override...", challenge.WaitSeconds)
			}
			return true, next.disableApprovedCmd(skipped, challenge.WaitSeconds)
		},
	}
	if skipped {
		m.modal.title = "Skip Tonight"
	}
	m.modal.form.WithWidth(maxInt(40, m.width-12))
	return nil
}

func (m model) disableApprovedCmd(skipped bool, waitSeconds int) tea.Cmd {
	return func() tea.Msg {
		if waitSeconds > 0 {
			time.Sleep(time.Duration(waitSeconds) * time.Second)
		}
		var (
			session schedule.Session
			err     error
		)
		if skipped {
			session, err = m.app.SkipTonightApproved(context.Background())
		} else {
			session, err = m.app.StopTonightApproved(context.Background())
		}
		if err != nil {
			return actionMsg{err: err}
		}
		return actionMsg{
			flash:  fmt.Sprintf("Updated curfew session %s.", session.Date),
			reload: true,
		}
	}
}

func huhFormForChallenge(challenge friction.Challenge, answer *string) *huh.Form {
	var fields []huh.Field
	fields = append(fields, huh.NewNote().
		Title(challenge.Reason).
		Description(challengeHelp(challenge)))
	if challenge.RequiresInput() {
		fields = append(fields, huh.NewInput().
			Title(challenge.Prompt).
			Value(answer))
	}
	form := huh.NewForm(huh.NewGroup(fields...)).
		WithShowHelp(false).
		WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }
	return form
}

func challengeHelp(challenge friction.Challenge) string {
	lines := []string{}
	if challenge.WaitSeconds > 0 {
		lines = append(lines, fmt.Sprintf("This override waits %d seconds before completing.", challenge.WaitSeconds))
	}
	if challenge.Help != "" {
		lines = append(lines, challenge.Help)
	}
	if len(lines) == 0 {
		return "Confirm to continue."
	}
	return strings.Join(lines, "\n")
}
