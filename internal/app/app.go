package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/friction"
	"github.com/iamrajjoshi/curfew/internal/paths"
	"github.com/iamrajjoshi/curfew/internal/rules"
	"github.com/iamrajjoshi/curfew/internal/runtime"
	"github.com/iamrajjoshi/curfew/internal/schedule"
	"github.com/iamrajjoshi/curfew/internal/store"
)

type App struct {
	Paths   paths.Layout
	Runtime *runtime.Manager
	Store   store.Store
	Now     func() time.Time
}

type CheckOptions struct {
	Shell string
	JSON  bool
	In    io.Reader
	Out   io.Writer
}

type CheckResult struct {
	Allowed      bool              `json:"allowed"`
	Reason       string            `json:"reason"`
	Tier         schedule.Tier     `json:"tier"`
	Outcome      string            `json:"outcome"`
	Action       string            `json:"action"`
	Match        rules.Match       `json:"match"`
	Session      *schedule.Session `json:"session,omitempty"`
	SnoozesLeft  int               `json:"snoozes_left"`
	SnoozedUntil string            `json:"snoozed_until,omitempty"`
}

type Status struct {
	Active       bool
	Forced       bool
	Disabled     bool
	SnoozedUntil string
	SnoozesLeft  int
	Tier         schedule.Tier
	Session      schedule.Session
	Next         schedule.Session
	Timezone     string
}

type DisableRequest struct {
	Target  schedule.Session
	Tier    schedule.Tier
	Profile friction.Profile
	Reason  string
}

func New() (*App, error) {
	layout, err := paths.Discover()
	if err != nil {
		return nil, err
	}
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	if err := layout.Ensure(); err != nil {
		return nil, err
	}

	var currentStore store.Store = store.Noop{}
	sqliteStore, err := store.Open(layout.HistoryDB())
	if err == nil {
		currentStore = sqliteStore
	}

	return &App{
		Paths:   layout,
		Runtime: runtime.New(layout.RuntimeFile(), layout.RuntimeLockFile()),
		Store:   currentStore,
		Now:     time.Now,
	}, nil
}

func (a *App) Close() error {
	return a.Store.Close()
}

func (a *App) LoadConfig() (config.Config, error) {
	return config.Load(a.Paths.ConfigFile())
}

func (a *App) HasConfig() bool {
	return config.Exists(a.Paths.ConfigFile())
}

func (a *App) SaveConfig(cfg config.Config) error {
	return config.Save(a.Paths.ConfigFile(), cfg)
}

func (a *App) DefaultConfig() config.Config {
	return config.Default()
}

func (a *App) EvaluateStatus() (Status, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return Status{}, err
	}
	now := a.Now()
	eval, err := schedule.Evaluate(now, cfg)
	if err != nil {
		return Status{}, err
	}
	runtimeState, err := a.Runtime.Read()
	if err != nil {
		return Status{}, err
	}

	session, tier, forced, disabled, snoozedUntil := effectiveSession(eval, runtimeState, now)
	snoozesLeft := cfg.Grace.SnoozeMaxPerNight
	if session != nil {
		snoozesLeft -= runtimeState.Sessions[session.Date].SnoozesUsed
	}
	if snoozesLeft < 0 {
		snoozesLeft = 0
	}

	status := Status{
		Active:       session != nil,
		Forced:       forced,
		Disabled:     disabled,
		SnoozedUntil: snoozedUntil,
		SnoozesLeft:  snoozesLeft,
		Tier:         tier,
		Next:         eval.Next,
		Timezone:     eval.Location.String(),
	}
	if session != nil {
		status.Session = *session
	}
	return status, nil
}

func (a *App) Check(ctx context.Context, raw string, opts CheckOptions) (CheckResult, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return CheckResult{
			Allowed: true,
			Reason:  "empty command",
			Tier:    schedule.TierNormal,
			Outcome: "allowed",
			Action:  "allow",
			Match:   rules.Match{Action: "allow"},
		}, nil
	}
	if !a.HasConfig() {
		return CheckResult{
			Allowed: true,
			Reason:  "curfew is not configured yet",
			Tier:    schedule.TierNormal,
			Outcome: "allowed",
			Action:  "allow",
			Match:   rules.Match{Action: "allow"},
		}, nil
	}

	cfg, err := a.LoadConfig()
	if err != nil {
		return CheckResult{}, err
	}

	now := a.Now()
	_ = a.Store.Purge(ctx, retentionStartDate(cfg, now, cfg.Logging.RetainDays))

	eval, err := schedule.Evaluate(now, cfg)
	if err != nil {
		return CheckResult{}, err
	}
	runtimeState, err := a.Runtime.Read()
	if err != nil {
		return CheckResult{}, err
	}
	currentSession, tier, _, disabled, snoozedUntil := effectiveSession(eval, runtimeState, now)
	match := rules.Evaluate(cfg, raw)

	result := CheckResult{
		Allowed:      true,
		Reason:       "allowed",
		Tier:         tier,
		Outcome:      "allowed",
		Action:       match.Action,
		Match:        match,
		Session:      currentSession,
		SnoozedUntil: snoozedUntil,
	}

	if currentSession == nil || disabled || match.Action == "allow" || match.AllowedByAllowlist {
		if currentSession != nil {
			result.SnoozesLeft = max(0, cfg.Grace.SnoozeMaxPerNight-runtimeState.Sessions[currentSession.Date].SnoozesUsed)
			a.logEvent(ctx, store.Event{
				SessionDate: currentSession.Date,
				Timestamp:   now,
				Command:     raw,
				MatchedRule: match.Pattern,
				Action:      match.Action,
				Outcome:     "allowed",
				Tier:        string(tier),
				Shell:       opts.Shell,
			}, *currentSession, runtimeState.Sessions[currentSession.Date], &now)
		}
		return result, nil
	}

	reason := buildReason(currentSession, tier, match)
	profile := friction.EffectiveProfile(cfg, match.Action, tier)
	allowed, outcome, err := friction.Run(profile, friction.IO{
		In:  opts.In,
		Out: opts.Out,
	}, reason)
	if err != nil {
		return CheckResult{}, err
	}

	result.Allowed = allowed
	result.Outcome = outcome
	result.Reason = reason
	result.SnoozesLeft = max(0, cfg.Grace.SnoozeMaxPerNight-runtimeState.Sessions[currentSession.Date].SnoozesUsed)

	firstBlocked := now
	record := store.SessionRecord{
		Date:              currentSession.Date,
		BedtimeConfigured: currentSession.BedtimeText,
		WakeConfigured:    currentSession.WakeText,
		LastCommandAt:     &now,
		SnoozesUsed:       runtimeState.Sessions[currentSession.Date].SnoozesUsed,
	}
	if !allowed {
		record.FirstBlockedAt = &firstBlocked
	}
	_ = a.Store.UpsertSession(ctx, record)
	_ = a.Store.RecordEvent(ctx, store.Event{
		SessionDate: currentSession.Date,
		Timestamp:   now,
		Command:     raw,
		MatchedRule: match.Pattern,
		Action:      match.Action,
		Outcome:     outcome,
		Tier:        string(tier),
		Shell:       opts.Shell,
	})

	return result, nil
}

func (a *App) SetForcedActive(ctx context.Context) (schedule.Session, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return schedule.Session{}, err
	}
	now := a.Now()
	eval, err := schedule.Evaluate(now, cfg)
	if err != nil {
		return schedule.Session{}, err
	}
	target := eval.Next
	if eval.Current != nil {
		target = *eval.Current
	}

	updated, err := a.Runtime.Update(now, func(file *runtime.File) error {
		state := file.Sessions[target.Date]
		state.ForcedActive = true
		state.Disabled = false
		state.SnoozedUntil = ""
		state.UpdatedAt = now.Format(time.RFC3339)
		file.Sessions[target.Date] = state
		return nil
	})
	if err != nil {
		return schedule.Session{}, err
	}
	_ = a.Store.UpsertSession(ctx, store.SessionRecord{
		Date:              target.Date,
		BedtimeConfigured: target.BedtimeText,
		WakeConfigured:    target.WakeText,
		SnoozesUsed:       updated.Sessions[target.Date].SnoozesUsed,
		ForcedActive:      true,
	})
	return target, nil
}

func (a *App) StopTonight(ctx context.Context, reason string, in io.Reader, out io.Writer) (schedule.Session, error) {
	return a.disableSession(ctx, false, reason, in, out)
}

func (a *App) SkipTonight(ctx context.Context, reason string, in io.Reader, out io.Writer) (schedule.Session, error) {
	return a.disableSession(ctx, true, reason, in, out)
}

func (a *App) ApplyDisableRequest(ctx context.Context, request DisableRequest, skipped bool, outcome string) (schedule.Session, error) {
	if request.Tier == schedule.TierHardStop || request.Profile.Kind == friction.KindBlock {
		return schedule.Session{}, errors.New("curfew cannot be disabled during hard stop")
	}
	return a.applyDisabledSession(ctx, request.Target, request.Tier, skipped, outcome)
}

func (a *App) Snooze(ctx context.Context, duration time.Duration) (schedule.Session, time.Time, int, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return schedule.Session{}, time.Time{}, 0, err
	}
	now := a.Now()
	eval, err := schedule.Evaluate(now, cfg)
	if err != nil {
		return schedule.Session{}, time.Time{}, 0, err
	}
	runtimeState, err := a.Runtime.Read()
	if err != nil {
		return schedule.Session{}, time.Time{}, 0, err
	}
	current, _, _, _, _ := effectiveSession(eval, runtimeState, now)
	if current == nil {
		return schedule.Session{}, time.Time{}, 0, errors.New("curfew is not active right now")
	}

	state := runtimeState.Sessions[current.Date]
	if state.SnoozesUsed >= cfg.Grace.SnoozeMaxPerNight {
		return schedule.Session{}, time.Time{}, 0, errors.New("no snoozes remaining tonight")
	}

	snoozedUntil := now.Add(duration)
	updated, err := a.Runtime.Update(now, func(file *runtime.File) error {
		entry := file.Sessions[current.Date]
		entry.SnoozesUsed++
		entry.SnoozedUntil = snoozedUntil.Format(time.RFC3339)
		entry.UpdatedAt = now.Format(time.RFC3339)
		file.Sessions[current.Date] = entry
		return nil
	})
	if err != nil {
		return schedule.Session{}, time.Time{}, 0, err
	}
	_ = a.Store.UpsertSession(ctx, store.SessionRecord{
		Date:              current.Date,
		BedtimeConfigured: current.BedtimeText,
		WakeConfigured:    current.WakeText,
		SnoozesUsed:       updated.Sessions[current.Date].SnoozesUsed,
	})
	remaining := max(0, cfg.Grace.SnoozeMaxPerNight-updated.Sessions[current.Date].SnoozesUsed)
	return *current, snoozedUntil, remaining, nil
}

func (a *App) History(ctx context.Context, days int) ([]store.HistoryRecord, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	now := a.Now()
	if err := a.materializeCompletedSessions(ctx, cfg, now, days); err != nil {
		return nil, err
	}
	return a.Store.History(ctx, historyStartDate(cfg, now, days))
}

func (a *App) Stats(ctx context.Context, days int) (store.Stats, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return store.Stats{}, err
	}
	now := a.Now()
	if err := a.materializeCompletedSessions(ctx, cfg, now, days); err != nil {
		return store.Stats{}, err
	}
	return a.Store.Stats(ctx, historyStartDate(cfg, now, days))
}

func (a *App) AddRule(pattern string, action string) error {
	cfg, err := a.LoadConfig()
	if err != nil {
		return err
	}
	cfg.Rules.Rule = append(cfg.Rules.Rule, config.RuleEntry{Pattern: pattern, Action: action})
	return a.SaveConfig(cfg)
}

func (a *App) RemoveRule(pattern string) (bool, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return false, err
	}
	filtered := make([]config.RuleEntry, 0, len(cfg.Rules.Rule))
	removed := false
	for _, rule := range cfg.Rules.Rule {
		if !removed && rule.Pattern == pattern {
			removed = true
			continue
		}
		filtered = append(filtered, rule)
	}
	if !removed {
		return false, nil
	}
	cfg.Rules.Rule = filtered
	return true, a.SaveConfig(cfg)
}

func (a *App) Rules() ([]config.RuleEntry, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Rules.Rule, nil
}

func (a *App) JSON(v any) string {
	bytes, _ := json.MarshalIndent(v, "", "  ")
	return string(bytes)
}

func (a *App) disableSession(ctx context.Context, skipped bool, reason string, in io.Reader, out io.Writer) (schedule.Session, error) {
	request, err := a.DisableRequest(skipped, reason)
	if err != nil {
		return schedule.Session{}, err
	}

	allowed, outcome, err := friction.Run(request.Profile, friction.IO{In: in, Out: out}, request.Reason)
	if err != nil {
		return schedule.Session{}, err
	}
	if !allowed {
		if request.Tier == schedule.TierHardStop {
			return schedule.Session{}, errors.New("curfew cannot be disabled during hard stop")
		}
		return schedule.Session{}, errors.New("override cancelled")
	}
	return a.ApplyDisableRequest(ctx, request, skipped, outcome)
}

func (a *App) DisableRequest(skipped bool, reason string) (DisableRequest, error) {
	cfg, err := a.LoadConfig()
	if err != nil {
		return DisableRequest{}, err
	}
	now := a.Now()
	eval, err := schedule.Evaluate(now, cfg)
	if err != nil {
		return DisableRequest{}, err
	}
	runtimeState, err := a.Runtime.Read()
	if err != nil {
		return DisableRequest{}, err
	}
	currentSession, tier, _, _, _ := effectiveSession(eval, runtimeState, now)
	target := eval.Next
	promptTier := schedule.TierCurfew
	if currentSession != nil {
		target = *currentSession
		promptTier = tier
	}

	return DisableRequest{
		Target:  target,
		Tier:    promptTier,
		Profile: friction.EffectiveProfile(cfg, "block", promptTier),
		Reason:  reason,
	}, nil
}

func (a *App) applyDisabledSession(ctx context.Context, target schedule.Session, tier schedule.Tier, skipped bool, outcome string) (schedule.Session, error) {
	now := a.Now()
	updated, err := a.Runtime.Update(now, func(file *runtime.File) error {
		state := file.Sessions[target.Date]
		state.Disabled = true
		state.ForcedActive = false
		state.SnoozedUntil = ""
		state.UpdatedAt = now.Format(time.RFC3339)
		file.Sessions[target.Date] = state
		return nil
	})
	if err != nil {
		return schedule.Session{}, err
	}

	record := store.SessionRecord{
		Date:              target.Date,
		BedtimeConfigured: target.BedtimeText,
		WakeConfigured:    target.WakeText,
		LastCommandAt:     &now,
		SnoozesUsed:       updated.Sessions[target.Date].SnoozesUsed,
		Skipped:           skipped,
	}
	_ = a.Store.UpsertSession(ctx, record)
	_ = a.Store.RecordEvent(ctx, store.Event{
		SessionDate: target.Date,
		Timestamp:   now,
		Command:     ternary(skipped, "curfew skip tonight", "curfew stop"),
		Action:      "override",
		Outcome:     outcome,
		Tier:        string(tier),
	})
	return target, nil
}

func buildReason(session *schedule.Session, tier schedule.Tier, match rules.Match) string {
	switch tier {
	case schedule.TierWarning:
		return fmt.Sprintf("Curfew warning window is active until bedtime at %s. \"%s\" matched %q.", session.Bedtime.Format("15:04"), match.CommandWord, match.Pattern)
	case schedule.TierHardStop:
		return fmt.Sprintf("Hard stop is active until wake at %s. \"%s\" matched %q.", session.Wake.Format("15:04"), match.CommandWord, match.Pattern)
	default:
		return fmt.Sprintf("Curfew is active until %s. \"%s\" matched %q.", session.Wake.Format("15:04"), match.CommandWord, match.Pattern)
	}
}

func effectiveSession(eval schedule.Evaluation, state runtime.File, now time.Time) (*schedule.Session, schedule.Tier, bool, bool, string) {
	session := eval.Current
	tier := eval.Tier
	forced := false
	disabled := false
	snoozedUntil := ""

	if session == nil {
		nextState := state.Sessions[eval.Next.Date]
		if nextState.ForcedActive {
			copy := eval.Next
			session = &copy
			tier = schedule.TierCurfew
			forced = true
		}
	}

	if session == nil {
		return nil, schedule.TierNormal, false, false, ""
	}

	sessionState := state.Sessions[session.Date]
	if sessionState.Disabled {
		return nil, schedule.TierNormal, forced, true, ""
	}
	if snoozed, ok := runtime.ParseTimestamp(sessionState.SnoozedUntil); ok && now.Before(snoozed) {
		return nil, schedule.TierNormal, forced, false, snoozed.Format(time.RFC3339)
	}
	if sessionState.ForcedActive && tier == schedule.TierNormal {
		tier = schedule.TierCurfew
		forced = true
	}
	return session, tier, forced, disabled, snoozedUntil
}

func (a *App) logEvent(ctx context.Context, event store.Event, session schedule.Session, state runtime.SessionState, lastCommandAt *time.Time) {
	_ = a.Store.RecordEvent(ctx, event)
	_ = a.Store.UpsertSession(ctx, store.SessionRecord{
		Date:              event.SessionDate,
		BedtimeConfigured: session.BedtimeText,
		WakeConfigured:    session.WakeText,
		LastCommandAt:     lastCommandAt,
		SnoozesUsed:       state.SnoozesUsed,
	})
}

func (a *App) materializeCompletedSessions(ctx context.Context, cfg config.Config, now time.Time, days int) error {
	location, err := schedule.ResolveLocation(cfg.Schedule.Timezone)
	if err != nil {
		return err
	}
	now = now.In(location)
	for offset := 0; offset <= max(days, 1); offset++ {
		date := time.Date(now.Year(), now.Month(), now.Day()-offset, 0, 0, 0, 0, location)
		session, err := schedule.SessionForDate(date, cfg, location)
		if err != nil {
			return err
		}
		if session.Wake.After(now) {
			continue
		}
		if err := a.Store.UpsertSession(ctx, store.SessionRecord{
			Date:              session.Date,
			BedtimeConfigured: session.BedtimeText,
			WakeConfigured:    session.WakeText,
		}); err != nil {
			return err
		}
	}
	return nil
}

func historyStartDate(cfg config.Config, now time.Time, days int) string {
	location, err := schedule.ResolveLocation(cfg.Schedule.Timezone)
	if err != nil {
		return now.Format("2006-01-02")
	}
	if days < 0 {
		days = 0
	}
	return now.In(location).AddDate(0, 0, -days).Format("2006-01-02")
}

func retentionStartDate(cfg config.Config, now time.Time, retainDays int) string {
	location, err := schedule.ResolveLocation(cfg.Schedule.Timezone)
	if err != nil {
		return now.Format("2006-01-02")
	}
	if retainDays < 0 {
		retainDays = 0
	}
	return now.In(location).AddDate(0, 0, -retainDays).Format("2006-01-02")
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func ternary[T any](condition bool, left T, right T) T {
	if condition {
		return left
	}
	return right
}

func (s Status) Render() string {
	var lines []string
	if s.Disabled {
		lines = append(lines, "Curfew disabled for this session.")
	} else if s.SnoozedUntil != "" {
		lines = append(lines, fmt.Sprintf("Curfew snoozed until %s.", s.SnoozedUntil))
	} else if s.Active {
		label := "Curfew active"
		if s.Forced {
			label = "Curfew force-enabled"
		}
		lines = append(lines, fmt.Sprintf("%s. %s until wake.", label, strings.ReplaceAll(string(s.Tier), "_", " ")))
		lines = append(lines, fmt.Sprintf("Bedtime: %s  Wake: %s  Hard stop: %s", s.Session.BedtimeText, s.Session.WakeText, s.Session.HardStopText))
	} else {
		lines = append(lines, fmt.Sprintf("Curfew inactive. Next bedtime is %s.", s.Next.Bedtime.Format("Mon 15:04")))
	}
	lines = append(lines, fmt.Sprintf("Snoozes left: %d", s.SnoozesLeft))
	lines = append(lines, fmt.Sprintf("Timezone: %s", s.Timezone))
	return strings.Join(lines, "\n")
}

func WriteCheckResult(result CheckResult, jsonMode bool, stderr io.Writer) {
	if jsonMode {
		bytes, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stderr, string(bytes))
	}
}

func StdIO() (io.Reader, io.Writer) {
	return os.Stdin, os.Stderr
}
