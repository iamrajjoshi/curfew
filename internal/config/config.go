package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

var weekdays = []string{
	"sunday",
	"monday",
	"tuesday",
	"wednesday",
	"thursday",
	"friday",
	"saturday",
}

type Config struct {
	Schedule  ScheduleConfig  `toml:"schedule"`
	Grace     GraceConfig     `toml:"grace"`
	Override  OverrideConfig  `toml:"override"`
	Rules     RulesConfig     `toml:"rules"`
	Allowlist AllowlistConfig `toml:"allowlist"`
	Logging   LoggingConfig   `toml:"logging"`
}

type ScheduleConfig struct {
	Timezone  string                 `toml:"timezone"`
	Default   DaySchedule            `toml:"default"`
	Overrides map[string]DaySchedule `toml:"overrides,omitempty"`
}

type DaySchedule struct {
	Bedtime string `toml:"bedtime"`
	Wake    string `toml:"wake"`
}

type GraceConfig struct {
	WarningWindow     string `toml:"warning_window"`
	HardStopAfter     string `toml:"hard_stop_after"`
	SnoozeMaxPerNight int    `toml:"snooze_max_per_night"`
	SnoozeDuration    string `toml:"snooze_duration"`
}

type OverrideConfig struct {
	Preset         string         `toml:"preset"`
	Method         string         `toml:"method,omitempty"`
	Passphrase     string         `toml:"passphrase,omitempty"`
	WaitSeconds    int            `toml:"wait_seconds,omitempty"`
	MathDifficulty string         `toml:"math_difficulty,omitempty"`
	Custom         CustomOverride `toml:"custom,omitempty"`
}

type CustomOverride struct {
	Warning  OverrideTier `toml:"warning,omitempty"`
	Curfew   OverrideTier `toml:"curfew,omitempty"`
	HardStop OverrideTier `toml:"hard_stop,omitempty"`
}

type OverrideTier struct {
	Action         string `toml:"action,omitempty"`
	Method         string `toml:"method,omitempty"`
	Passphrase     string `toml:"passphrase,omitempty"`
	WaitSeconds    int    `toml:"wait_seconds,omitempty"`
	MathDifficulty string `toml:"math_difficulty,omitempty"`
}

type RulesConfig struct {
	DefaultAction string      `toml:"default_action"`
	Rule          []RuleEntry `toml:"rule"`
}

type RuleEntry struct {
	Pattern string `toml:"pattern"`
	Action  string `toml:"action"`
}

type AllowlistConfig struct {
	Always []string `toml:"always"`
}

type LoggingConfig struct {
	RetainDays int `toml:"retain_days"`
}

func Default() Config {
	return Config{
		Schedule: ScheduleConfig{
			Timezone: "auto",
			Default: DaySchedule{
				Bedtime: "23:00",
				Wake:    "07:00",
			},
			Overrides: map[string]DaySchedule{
				"friday": {
					Bedtime: "00:00",
					Wake:    "09:00",
				},
				"saturday": {
					Bedtime: "00:00",
					Wake:    "09:00",
				},
			},
		},
		Grace: GraceConfig{
			WarningWindow:     "30m",
			HardStopAfter:     "01:00",
			SnoozeMaxPerNight: 2,
			SnoozeDuration:    "15m",
		},
		Override: OverrideConfig{
			Preset:         "medium",
			Method:         "passphrase",
			Passphrase:     "i am choosing to break my own rule",
			WaitSeconds:    60,
			MathDifficulty: "medium",
		},
		Rules: RulesConfig{
			DefaultAction: "allow",
			Rule: []RuleEntry{
				{Pattern: "claude", Action: "block"},
				{Pattern: "cursor-agent", Action: "block"},
				{Pattern: "aider*", Action: "block"},
				{Pattern: "git push*", Action: "warn"},
				{Pattern: "git commit*", Action: "warn"},
				{Pattern: "terraform apply*", Action: "block"},
				{Pattern: "npm run deploy*", Action: "block"},
				{Pattern: "kubectl apply*", Action: "block"},
			},
		},
		Allowlist: AllowlistConfig{
			Always: []string{
				"ls",
				"cd",
				"cat",
				"less",
				"man",
				"tldr",
				"curfew",
				"exit",
				"clear",
			},
		},
		Logging: LoggingConfig{
			RetainDays: 90,
		},
	}
}

func Load(path string) (Config, error) {
	var cfg Config
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := toml.Unmarshal(bytes, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	bytes, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o644)
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Clone(cfg Config) Config {
	copy := cfg
	copy.Schedule.Overrides = cloneOverrides(cfg.Schedule.Overrides)
	copy.Rules.Rule = append([]RuleEntry(nil), cfg.Rules.Rule...)
	copy.Allowlist.Always = append([]string(nil), cfg.Allowlist.Always...)
	return copy
}

func (c Config) Validate() error {
	if err := c.Schedule.Validate(); err != nil {
		return err
	}
	if err := c.Grace.Validate(); err != nil {
		return err
	}
	if err := c.Override.Validate(); err != nil {
		return err
	}
	if err := c.Rules.Validate(); err != nil {
		return err
	}
	if c.Logging.RetainDays <= 0 {
		return fmt.Errorf("logging.retain_days must be positive")
	}
	return nil
}

func (c ScheduleConfig) Validate() error {
	if strings.TrimSpace(c.Timezone) == "" {
		return fmt.Errorf("schedule.timezone is required")
	}
	if err := c.Default.Validate("schedule.default"); err != nil {
		return err
	}
	for day, schedule := range c.Overrides {
		if !slices.Contains(weekdays, strings.ToLower(day)) {
			return fmt.Errorf("schedule.overrides.%s is not a valid weekday", day)
		}
		if err := schedule.Validate("schedule.overrides." + day); err != nil {
			return err
		}
	}
	return nil
}

func (d DaySchedule) Validate(path string) error {
	if _, err := parseClock(d.Bedtime); err != nil {
		return fmt.Errorf("%s.bedtime: %w", path, err)
	}
	if _, err := parseClock(d.Wake); err != nil {
		return fmt.Errorf("%s.wake: %w", path, err)
	}
	return nil
}

func (g GraceConfig) Validate() error {
	if _, err := time.ParseDuration(g.WarningWindow); err != nil {
		return fmt.Errorf("grace.warning_window: %w", err)
	}
	if _, err := time.ParseDuration(g.SnoozeDuration); err != nil {
		return fmt.Errorf("grace.snooze_duration: %w", err)
	}
	if _, err := parseClock(g.HardStopAfter); err != nil {
		return fmt.Errorf("grace.hard_stop_after: %w", err)
	}
	if g.SnoozeMaxPerNight < 0 {
		return fmt.Errorf("grace.snooze_max_per_night must be >= 0")
	}
	return nil
}

func (o OverrideConfig) Validate() error {
	preset := strings.ToLower(strings.TrimSpace(o.Preset))
	if !slices.Contains([]string{"soft", "medium", "hard", "custom"}, preset) {
		return fmt.Errorf("override.preset must be one of soft, medium, hard, custom")
	}
	if preset != "custom" {
		return nil
	}
	if err := o.Custom.Warning.Validate("override.custom.warning"); err != nil {
		return err
	}
	if err := o.Custom.Curfew.Validate("override.custom.curfew"); err != nil {
		return err
	}
	if err := o.Custom.HardStop.Validate("override.custom.hard_stop"); err != nil {
		return err
	}
	return nil
}

func (o OverrideTier) Validate(path string) error {
	action := strings.TrimSpace(strings.ToLower(o.Action))
	method := strings.TrimSpace(strings.ToLower(o.Method))
	if action != "" && !slices.Contains([]string{"allow", "block", "friction"}, action) {
		return fmt.Errorf("%s.action must be one of allow, block, friction", path)
	}
	if method != "" && !slices.Contains([]string{"none", "prompt", "passphrase", "math", "wait", "combined"}, method) {
		return fmt.Errorf("%s.method must be one of none, prompt, passphrase, math, wait, combined", path)
	}
	if o.WaitSeconds < 0 {
		return fmt.Errorf("%s.wait_seconds must be >= 0", path)
	}
	if difficulty := strings.TrimSpace(strings.ToLower(o.MathDifficulty)); difficulty != "" &&
		!slices.Contains([]string{"easy", "medium", "hard"}, difficulty) {
		return fmt.Errorf("%s.math_difficulty must be one of easy, medium, hard", path)
	}
	return nil
}

func (r RulesConfig) Validate() error {
	defaultAction := strings.ToLower(strings.TrimSpace(r.DefaultAction))
	if !slices.Contains([]string{"allow", "block", "warn", "delay"}, defaultAction) {
		return fmt.Errorf("rules.default_action must be one of allow, block, warn, delay")
	}
	for i, rule := range r.Rule {
		if strings.TrimSpace(rule.Pattern) == "" {
			return fmt.Errorf("rules.rule[%d].pattern is required", i)
		}
		action := strings.ToLower(strings.TrimSpace(rule.Action))
		if !slices.Contains([]string{"allow", "block", "warn", "delay"}, action) {
			return fmt.Errorf("rules.rule[%d].action must be one of allow, block, warn, delay", i)
		}
		if strings.Count(rule.Pattern, "*") > 1 || (strings.Contains(rule.Pattern, "*") && !strings.HasSuffix(rule.Pattern, "*")) {
			return fmt.Errorf("rules.rule[%d].pattern only supports a single trailing *", i)
		}
	}
	return nil
}

func parseClock(value string) (time.Duration, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("must be in HH:MM format")
	}
	hour, err := time.Parse("15", parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour %q", parts[0])
	}
	minute, err := time.Parse("04", parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute %q", parts[1])
	}
	return time.Duration(hour.Hour())*time.Hour + time.Duration(minute.Minute())*time.Minute, nil
}

func WeekdayKey(day time.Weekday) string {
	return weekdays[day]
}

func cloneOverrides(input map[string]DaySchedule) map[string]DaySchedule {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]DaySchedule, len(input))
	for day, schedule := range input {
		output[day] = schedule
	}
	return output
}
