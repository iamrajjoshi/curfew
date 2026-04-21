package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/friction"
	"github.com/iamrajjoshi/curfew/internal/schedule"
)

func (t overrideTab) update(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "p", "e":
		m.openOverrideBaseModal()
	case "1":
		m.openOverrideTierModal("warning")
	case "2":
		m.openOverrideTierModal("curfew")
	case "3":
		m.openOverrideTierModal("hard_stop")
	}
	return m, nil
}

func (t overrideTab) view(m model) string {
	lines := []string{"Override", "--------"}
	lines = append(lines, fmt.Sprintf("Preset: %s", m.draft.Override.Preset))
	lines = append(lines, fmt.Sprintf("Passphrase: %s", m.draft.Override.Passphrase))
	lines = append(lines, fmt.Sprintf("Wait seconds: %d", m.draft.Override.WaitSeconds))
	lines = append(lines, fmt.Sprintf("Math difficulty: %s", m.draft.Override.MathDifficulty))
	lines = append(lines, "")
	lines = append(lines, "Blocked-command preview by tier:")
	for _, tier := range []schedule.Tier{schedule.TierWarning, schedule.TierCurfew, schedule.TierHardStop} {
		profile := friction.EffectiveProfile(m.draft, "block", tier)
		lines = append(lines, fmt.Sprintf("  %-9s %s", tier, describeProfile(profile)))
	}
	lines = append(lines, "")
	lines = append(lines, "Custom tier settings:")
	lines = append(lines, fmt.Sprintf("  [1] warning  %s", describeTier(m.draft.Override.Custom.Warning)))
	lines = append(lines, fmt.Sprintf("  [2] curfew   %s", describeTier(m.draft.Override.Custom.Curfew)))
	lines = append(lines, fmt.Sprintf("  [3] hardstop %s", describeTier(m.draft.Override.Custom.HardStop)))
	lines = append(lines, "")
	lines = append(lines, "Keys: [p] edit preset/base settings  [1]/[2]/[3] edit custom tiers")
	return strings.Join(lines, "\n")
}

func (m *model) openOverrideBaseModal() {
	preset := m.draft.Override.Preset
	passphrase := m.draft.Override.Passphrase
	waitSeconds := strconv.Itoa(m.draft.Override.WaitSeconds)
	mathDifficulty := m.draft.Override.MathDifficulty

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Preset").
				Options(
					huh.NewOption("Soft", "soft"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("Hard", "hard"),
					huh.NewOption("Custom", "custom"),
				).
				Value(&preset),
			huh.NewInput().Title("Passphrase").Value(&passphrase),
			huh.NewInput().Title("Wait seconds").Value(&waitSeconds),
			huh.NewSelect[string]().
				Title("Math difficulty").
				Options(
					huh.NewOption("Easy", "easy"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("Hard", "hard"),
				).
				Value(&mathDifficulty),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	modal := &modalState{
		title: "Edit Override Settings",
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			parsed, err := strconv.Atoi(strings.TrimSpace(waitSeconds))
			if err != nil {
				next.flash = "Wait seconds must be a number."
				return false, nil
			}
			next.draft.Override.Preset = strings.TrimSpace(strings.ToLower(preset))
			next.draft.Override.Passphrase = strings.TrimSpace(passphrase)
			next.draft.Override.WaitSeconds = parsed
			next.draft.Override.MathDifficulty = strings.TrimSpace(strings.ToLower(mathDifficulty))
			next.flash = "Updated override settings in the draft config."
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}

func (m *model) openOverrideTierModal(kind string) {
	target := selectTier(m.draft, kind)
	action := target.Action
	method := target.Method
	passphrase := target.Passphrase
	waitSeconds := strconv.Itoa(target.WaitSeconds)
	mathDifficulty := target.MathDifficulty

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Action").
				Options(
					huh.NewOption("Inherit / none", ""),
					huh.NewOption("Allow", "allow"),
					huh.NewOption("Block", "block"),
					huh.NewOption("Friction", "friction"),
				).
				Value(&action),
			huh.NewSelect[string]().
				Title("Method").
				Options(
					huh.NewOption("Inherit / none", ""),
					huh.NewOption("None", "none"),
					huh.NewOption("Prompt", "prompt"),
					huh.NewOption("Passphrase", "passphrase"),
					huh.NewOption("Math", "math"),
					huh.NewOption("Wait", "wait"),
					huh.NewOption("Combined", "combined"),
				).
				Value(&method),
			huh.NewInput().Title("Passphrase").Value(&passphrase),
			huh.NewInput().Title("Wait seconds").Value(&waitSeconds),
			huh.NewSelect[string]().
				Title("Math difficulty").
				Options(
					huh.NewOption("Inherit", ""),
					huh.NewOption("Easy", "easy"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("Hard", "hard"),
				).
				Value(&mathDifficulty),
		),
	).WithShowHelp(false).WithShowErrors(true)
	form.SubmitCmd = func() tea.Msg { return modalSubmittedMsg{} }
	form.CancelCmd = func() tea.Msg { return modalCancelledMsg{} }

	modal := &modalState{
		title: fmt.Sprintf("Edit %s Tier", titleCase(kind)),
		form:  form,
		apply: func(next *model) (bool, tea.Cmd) {
			parsed, err := strconv.Atoi(strings.TrimSpace(waitSeconds))
			if err != nil {
				next.flash = "Wait seconds must be a number."
				return false, nil
			}
			tier := config.OverrideTier{
				Action:         strings.TrimSpace(strings.ToLower(action)),
				Method:         strings.TrimSpace(strings.ToLower(method)),
				Passphrase:     strings.TrimSpace(passphrase),
				WaitSeconds:    parsed,
				MathDifficulty: strings.TrimSpace(strings.ToLower(mathDifficulty)),
			}
			assignTier(&next.draft, kind, tier)
			next.flash = fmt.Sprintf("Updated %s tier in the draft config.", kind)
			return true, nil
		},
	}
	modal.form.WithWidth(maxInt(40, m.width-12))
	m.modal = modal
}

func selectTier(cfg config.Config, kind string) config.OverrideTier {
	switch kind {
	case "warning":
		return cfg.Override.Custom.Warning
	case "curfew":
		return cfg.Override.Custom.Curfew
	case "hard_stop":
		return cfg.Override.Custom.HardStop
	default:
		return config.OverrideTier{}
	}
}

func assignTier(cfg *config.Config, kind string, tier config.OverrideTier) {
	switch kind {
	case "warning":
		cfg.Override.Custom.Warning = tier
	case "curfew":
		cfg.Override.Custom.Curfew = tier
	case "hard_stop":
		cfg.Override.Custom.HardStop = tier
	}
}

func describeTier(tier config.OverrideTier) string {
	if tier == (config.OverrideTier{}) {
		return "inherit preset behavior"
	}
	parts := []string{}
	if tier.Action != "" {
		parts = append(parts, "action="+tier.Action)
	}
	if tier.Method != "" {
		parts = append(parts, "method="+tier.Method)
	}
	if tier.WaitSeconds > 0 {
		parts = append(parts, fmt.Sprintf("wait=%ds", tier.WaitSeconds))
	}
	if tier.MathDifficulty != "" {
		parts = append(parts, "math="+tier.MathDifficulty)
	}
	if len(parts) == 0 {
		return "inherit preset behavior"
	}
	return strings.Join(parts, ", ")
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

func titleCase(value string) string {
	replacer := strings.NewReplacer("_", " ", "-", " ")
	parts := strings.Fields(replacer.Replace(value))
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
