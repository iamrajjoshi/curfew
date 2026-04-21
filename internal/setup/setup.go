package setup

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/iamrajjoshi/curfew/internal/config"
)

func Run(current config.Config) (config.Config, error) {
	cfg := current

	bedtime := cfg.Schedule.Default.Bedtime
	wake := cfg.Schedule.Default.Wake
	timezone := cfg.Schedule.Timezone
	preset := cfg.Override.Preset
	passphrase := cfg.Override.Passphrase
	fridayBedtime := cfg.Schedule.Overrides["friday"].Bedtime
	fridayWake := cfg.Schedule.Overrides["friday"].Wake
	saturdayBedtime := cfg.Schedule.Overrides["saturday"].Bedtime
	saturdayWake := cfg.Schedule.Overrides["saturday"].Wake

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Weeknight bedtime (HH:MM)").
				Description("Example: 23:00").
				Value(&bedtime),
			huh.NewInput().
				Title("Weekday wake time (HH:MM)").
				Description("Example: 07:00").
				Value(&wake),
			huh.NewInput().
				Title("Timezone").
				Description("Use auto to follow the system timezone. Otherwise enter an IANA name like America/Los_Angeles.").
				Value(&timezone),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Friday bedtime (HH:MM)").
				Value(&fridayBedtime),
			huh.NewInput().
				Title("Friday wake time (HH:MM)").
				Value(&fridayWake),
			huh.NewInput().
				Title("Saturday bedtime (HH:MM)").
				Value(&saturdayBedtime),
			huh.NewInput().
				Title("Saturday wake time (HH:MM)").
				Value(&saturdayWake),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Override preset").
				Options(
					huh.NewOption("Soft", "soft"),
					huh.NewOption("Medium", "medium"),
					huh.NewOption("Hard", "hard"),
					huh.NewOption("Custom", "custom"),
				).
				Value(&preset),
			huh.NewInput().
				Title("Passphrase override").
				Description("Used by the medium and hard presets, and by custom passphrase flows.").
				Value(&passphrase),
		),
	)

	if err := form.Run(); err != nil {
		return config.Config{}, err
	}

	cfg.Schedule.Default.Bedtime = strings.TrimSpace(bedtime)
	cfg.Schedule.Default.Wake = strings.TrimSpace(wake)
	cfg.Schedule.Timezone = strings.TrimSpace(timezone)
	if cfg.Schedule.Timezone == "" {
		cfg.Schedule.Timezone = "auto"
	}
	cfg.Schedule.Overrides = map[string]config.DaySchedule{
		"friday": {
			Bedtime: strings.TrimSpace(fridayBedtime),
			Wake:    strings.TrimSpace(fridayWake),
		},
		"saturday": {
			Bedtime: strings.TrimSpace(saturdayBedtime),
			Wake:    strings.TrimSpace(saturdayWake),
		},
	}
	cfg.Override.Preset = strings.TrimSpace(strings.ToLower(preset))
	cfg.Override.Passphrase = strings.TrimSpace(passphrase)

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("invalid answers: %w", err)
	}
	return cfg, nil
}
