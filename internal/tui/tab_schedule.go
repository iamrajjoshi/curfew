package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/iamrajjoshi/curfew/internal/config"
)

var weekdayOrder = []string{
	"monday",
	"tuesday",
	"wednesday",
	"thursday",
	"friday",
	"saturday",
	"sunday",
}

func (t scheduleTab) update(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		if t.selectedDay > 0 {
			t.selectedDay--
		}
		m.scheduleTab = t
	case "down", "j":
		if t.selectedDay < len(weekdayOrder)-1 {
			t.selectedDay++
		}
		m.scheduleTab = t
	case "g":
		m.openScheduleGeneralModal()
	case "e", "enter":
		m.openScheduleDayModal(weekdayOrder[t.selectedDay])
	case "d":
		day := weekdayOrder[t.selectedDay]
		delete(m.draft.Schedule.Overrides, day)
		m.syncDraftState()
		m.flash = fmt.Sprintf("Cleared %s override from the draft config.", day)
	}

	return m, nil
}

func (t scheduleTab) view(m model) string {
	lines := []string{"Schedule", "--------"}
	lines = append(lines, fmt.Sprintf("Timezone: %s", m.draft.Schedule.Timezone))
	lines = append(lines, fmt.Sprintf("Default: bedtime %s -> wake %s", m.draft.Schedule.Default.Bedtime, m.draft.Schedule.Default.Wake))
	lines = append(lines, fmt.Sprintf("Warning window: %s", m.draft.Grace.WarningWindow))
	lines = append(lines, fmt.Sprintf("Hard stop: %s", m.draft.Grace.HardStopAfter))
	lines = append(lines, fmt.Sprintf("Snoozes per night: %d", m.draft.Grace.SnoozeMaxPerNight))
	lines = append(lines, fmt.Sprintf("Snooze duration: %s", m.draft.Grace.SnoozeDuration))
	lines = append(lines, "")
	lines = append(lines, "Weekday overrides:")
	for i, day := range weekdayOrder {
		cursor := " "
		if i == t.selectedDay {
			cursor = ">"
		}
		override, ok := m.draft.Schedule.Overrides[day]
		if ok {
			lines = append(lines, fmt.Sprintf("%s %-9s bedtime %s -> wake %s", cursor, day, override.Bedtime, override.Wake))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %-9s default schedule", cursor, day))
	}
	lines = append(lines, "")
	lines = append(lines, "Keys: [g] edit general schedule  [e] edit selected day  [d] clear selected override")
	return strings.Join(lines, "\n")
}

func (m *model) openScheduleGeneralModal() {
	bedtime := m.draft.Schedule.Default.Bedtime
	wake := m.draft.Schedule.Default.Wake
	timezone := m.draft.Schedule.Timezone
	warningWindow := m.draft.Grace.WarningWindow
	hardStop := m.draft.Grace.HardStopAfter
	snoozeMax := strconv.Itoa(m.draft.Grace.SnoozeMaxPerNight)
	snoozeDuration := m.draft.Grace.SnoozeDuration

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Default bedtime (HH:MM)").Value(&bedtime),
			huh.NewInput().Title("Default wake time (HH:MM)").Value(&wake),
			huh.NewInput().Title("Timezone").Value(&timezone),
			huh.NewInput().Title("Warning window").Value(&warningWindow),
			huh.NewInput().Title("Hard stop (HH:MM)").Value(&hardStop),
			huh.NewInput().Title("Snoozes per night").Value(&snoozeMax),
			huh.NewInput().Title("Snooze duration").Value(&snoozeDuration),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	modal := &modalState{
		title: "Edit Schedule",
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			parsed, err := strconv.Atoi(strings.TrimSpace(snoozeMax))
			if err != nil {
				next.flash = "Snoozes per night must be a number."
				return false, nil
			}
			next.draft.Schedule.Default = config.DaySchedule{
				Bedtime: strings.TrimSpace(bedtime),
				Wake:    strings.TrimSpace(wake),
			}
			next.draft.Schedule.Timezone = strings.TrimSpace(timezone)
			if next.draft.Schedule.Timezone == "" {
				next.draft.Schedule.Timezone = "auto"
			}
			next.draft.Grace.WarningWindow = strings.TrimSpace(warningWindow)
			next.draft.Grace.HardStopAfter = strings.TrimSpace(hardStop)
			next.draft.Grace.SnoozeMaxPerNight = parsed
			next.draft.Grace.SnoozeDuration = strings.TrimSpace(snoozeDuration)
			next.flash = "Updated schedule settings in the draft config."
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}

func (m *model) openScheduleDayModal(day string) {
	current, ok := m.draft.Schedule.Overrides[day]
	if !ok {
		current = m.draft.Schedule.Default
	}
	bedtime := current.Bedtime
	wake := current.Wake

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(fmt.Sprintf("%s bedtime (HH:MM)", strings.Title(day))).Value(&bedtime),
			huh.NewInput().Title(fmt.Sprintf("%s wake time (HH:MM)", strings.Title(day))).Value(&wake),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	modal := &modalState{
		title: fmt.Sprintf("Edit %s Override", strings.Title(day)),
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			if next.draft.Schedule.Overrides == nil {
				next.draft.Schedule.Overrides = map[string]config.DaySchedule{}
			}
			next.draft.Schedule.Overrides[day] = config.DaySchedule{
				Bedtime: strings.TrimSpace(bedtime),
				Wake:    strings.TrimSpace(wake),
			}
			next.flash = fmt.Sprintf("Updated %s override in the draft config.", day)
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}
