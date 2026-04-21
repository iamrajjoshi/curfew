package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iamrajjoshi/curfew/internal/config"
	"github.com/iamrajjoshi/curfew/internal/paths"
	"github.com/iamrajjoshi/curfew/internal/runtime"
	"github.com/iamrajjoshi/curfew/internal/store"
)

func TestCheckHistoryStatsAndSnooze(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 23, 23, 30, 0, 0, location)
	application, clock := newTestApp(t, now)

	blocked, err := application.Check(context.Background(), "claude", CheckOptions{
		In:  strings.NewReader("nope\n"),
		Out: &strings.Builder{},
	})
	if err != nil {
		t.Fatalf("check blocked: %v", err)
	}
	if blocked.Allowed {
		t.Fatal("expected claude to be blocked with the wrong passphrase")
	}
	if blocked.Outcome != "blocked" {
		t.Fatalf("blocked outcome = %q, want blocked", blocked.Outcome)
	}

	allowed, err := application.Check(context.Background(), "claude", CheckOptions{
		In:  strings.NewReader("i am choosing to break my own rule\n"),
		Out: &strings.Builder{},
	})
	if err != nil {
		t.Fatalf("check allowed: %v", err)
	}
	if !allowed.Allowed {
		t.Fatal("expected claude to be allowed with the correct passphrase")
	}
	if allowed.Outcome != "overridden" {
		t.Fatalf("allowed outcome = %q, want overridden", allowed.Outcome)
	}

	history, err := application.History(context.Background(), 7)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	record := findHistoryRecord(t, history, "2026-04-23")
	if record.Status != "overrode" {
		t.Fatalf("history status = %q, want overrode", record.Status)
	}

	stats, err := application.Stats(context.Background(), 7)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalNights < 1 {
		t.Fatalf("total nights = %d, want at least 1", stats.TotalNights)
	}
	if stats.AdherentNights != stats.RespectedNights+stats.SnoozedNights {
		t.Fatalf("adherent nights = %d, want %d", stats.AdherentNights, stats.RespectedNights+stats.SnoozedNights)
	}
	if stats.OverriddenNights < 1 {
		t.Fatalf("overridden nights = %d, want at least 1", stats.OverriddenNights)
	}
	if stats.MostAttemptedCommand != "claude" {
		t.Fatalf("most attempted command = %q, want claude", stats.MostAttemptedCommand)
	}
	if len(stats.TopCommands) == 0 || stats.TopCommands[0].Command != "claude" {
		t.Fatalf("top commands = %+v, want claude first", stats.TopCommands)
	}

	session, snoozedUntil, remaining, err := application.Snooze(context.Background(), 15*time.Minute)
	if err != nil {
		t.Fatalf("snooze: %v", err)
	}
	if session.Date != "2026-04-23" {
		t.Fatalf("session date = %q, want 2026-04-23", session.Date)
	}
	if remaining != 1 {
		t.Fatalf("remaining snoozes = %d, want 1", remaining)
	}
	if got := snoozedUntil.In(location).Format(time.RFC3339); got != "2026-04-23T23:45:00-07:00" {
		t.Fatalf("snoozed until = %q, want 2026-04-23T23:45:00-07:00", got)
	}

	status, err := application.EvaluateStatus()
	if err != nil {
		t.Fatalf("evaluate status while snoozed: %v", err)
	}
	if status.SnoozedUntil == "" {
		t.Fatal("expected status to show an active snooze")
	}

	clock.now = clock.now.Add(20 * time.Minute)
	status, err = application.EvaluateStatus()
	if err != nil {
		t.Fatalf("evaluate status after snooze: %v", err)
	}
	if !status.Active {
		t.Fatal("expected curfew to become active again after snooze expiry")
	}
}

func TestStopAndSkipDisableTheSession(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 22, 23, 30, 0, 0, location)
	application, clock := newTestApp(t, now)

	stopSession, err := application.StopTonight(
		context.Background(),
		"Disable curfew for the rest of this session?",
		strings.NewReader("i am choosing to break my own rule\n"),
		&strings.Builder{},
	)
	if err != nil {
		t.Fatalf("stop tonight: %v", err)
	}
	if stopSession.Date != "2026-04-22" {
		t.Fatalf("stop session = %q, want 2026-04-22", stopSession.Date)
	}

	status, err := application.EvaluateStatus()
	if err != nil {
		t.Fatalf("evaluate status after stop: %v", err)
	}
	if !status.Disabled {
		t.Fatal("expected stop tonight to disable the current session")
	}
	history, err := application.History(context.Background(), 7)
	if err != nil {
		t.Fatalf("history after stop: %v", err)
	}
	if record := findHistoryRecord(t, history, "2026-04-22"); record.Status != "overrode" {
		t.Fatalf("expected stop to register as an override, got %+v", record)
	}

	clock.now = time.Date(2026, 4, 23, 23, 45, 0, 0, location)
	skipSession, err := application.SkipTonight(
		context.Background(),
		"Skip tonight's curfew?",
		strings.NewReader("i am choosing to break my own rule\n"),
		&strings.Builder{},
	)
	if err != nil {
		t.Fatalf("skip tonight: %v", err)
	}
	if skipSession.Date != "2026-04-23" {
		t.Fatalf("skip session = %q, want 2026-04-23", skipSession.Date)
	}

	history, err = application.History(context.Background(), 7)
	if err != nil {
		t.Fatalf("history after skip: %v", err)
	}
	if record := findHistoryRecord(t, history, "2026-04-23"); record.Status != "overrode" {
		t.Fatalf("expected skip to register as an override, got %+v", record)
	}
}

func TestHardStopPreventsStopOverride(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 25, 1, 30, 0, 0, location)
	application, _ := newTestApp(t, now)

	_, err = application.StopTonight(
		context.Background(),
		"Disable curfew for the rest of this session?",
		strings.NewReader("i am choosing to break my own rule\n"),
		&strings.Builder{},
	)
	if err == nil {
		t.Fatal("expected hard stop to reject the stop override")
	}
	if !strings.Contains(err.Error(), "hard stop") {
		t.Fatalf("expected a hard-stop error, got %v", err)
	}

	status, err := application.EvaluateStatus()
	if err != nil {
		t.Fatalf("evaluate status after failed stop: %v", err)
	}
	if status.Disabled {
		t.Fatal("hard stop should not disable the session")
	}
}

func TestApplyDisableRequestUsesOriginalTarget(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 22, 23, 30, 0, 0, location)
	application, clock := newTestApp(t, now)

	request, err := application.DisableRequest(false, "Disable curfew for the rest of this session?")
	if err != nil {
		t.Fatalf("build disable request: %v", err)
	}
	if request.Target.Date != "2026-04-22" {
		t.Fatalf("request target date = %q, want 2026-04-22", request.Target.Date)
	}

	clock.now = time.Date(2026, 4, 23, 7, 30, 0, 0, location)
	session, err := application.ApplyDisableRequest(context.Background(), request, false, "overridden")
	if err != nil {
		t.Fatalf("apply disable request: %v", err)
	}
	if session.Date != "2026-04-22" {
		t.Fatalf("applied session date = %q, want 2026-04-22", session.Date)
	}

	history, err := application.History(context.Background(), 7)
	if err != nil {
		t.Fatalf("history after apply: %v", err)
	}
	if record := findHistoryRecord(t, history, "2026-04-22"); record.Status != "overrode" {
		t.Fatalf("expected original session to register as overrode, got %+v", record)
	}
}

func TestQuietCompletedSessionsAppearInHistoryAndStats(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, location)
	application, _ := newTestApp(t, now)

	history, err := application.History(context.Background(), 2)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if record := findHistoryRecord(t, history, "2026-04-20"); record.Status != "respected" {
		t.Fatalf("expected 2026-04-20 to be respected, got %+v", record)
	}
	if record := findHistoryRecord(t, history, "2026-04-19"); record.Status != "respected" {
		t.Fatalf("expected 2026-04-19 to be respected, got %+v", record)
	}

	stats, err := application.Stats(context.Background(), 2)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.RespectedNights < 2 {
		t.Fatalf("respected nights = %d, want at least 2", stats.RespectedNights)
	}
}

func TestHistoryDetailsIncludeOrderedEvents(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 23, 23, 30, 0, 0, location)
	application, _ := newTestApp(t, now)

	_, err = application.Check(context.Background(), "claude", CheckOptions{
		Shell: "zsh",
		In:    strings.NewReader("nope\n"),
		Out:   &strings.Builder{},
	})
	if err != nil {
		t.Fatalf("blocked check: %v", err)
	}
	_, err = application.Check(context.Background(), "claude", CheckOptions{
		Shell: "zsh",
		In:    strings.NewReader("i am choosing to break my own rule\n"),
		Out:   &strings.Builder{},
	})
	if err != nil {
		t.Fatalf("allowed check: %v", err)
	}

	details, err := application.HistoryDetails(context.Background(), "2026-04-23")
	if err != nil {
		t.Fatalf("history details: %v", err)
	}
	if !details.Found {
		t.Fatal("expected history details to be found")
	}
	if details.Session.Status != "overrode" {
		t.Fatalf("session status = %q, want overrode", details.Session.Status)
	}
	if len(details.Events) != 2 {
		t.Fatalf("event count = %d, want 2", len(details.Events))
	}
	if details.Events[0].Outcome != "blocked" || details.Events[1].Outcome != "overridden" {
		t.Fatalf("event outcomes = %+v, want blocked then overridden", details.Events)
	}
	if details.Events[0].Shell != "zsh" || details.Events[0].MatchedRule != "claude" {
		t.Fatalf("first event = %+v, want shell zsh and matched rule claude", details.Events[0])
	}
}

func TestHistoryDetailsMaterializeQuietNight(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, location)
	application, _ := newTestApp(t, now)

	details, err := application.HistoryDetails(context.Background(), "2026-04-20")
	if err != nil {
		t.Fatalf("history details: %v", err)
	}
	if !details.Found {
		t.Fatal("expected quiet night details to be found")
	}
	if details.Session.Status != "respected" {
		t.Fatalf("session status = %q, want respected", details.Session.Status)
	}
	if len(details.Events) != 0 {
		t.Fatalf("quiet night events = %+v, want none", details.Events)
	}
}

func findHistoryRecord(t *testing.T, history []store.HistoryRecord, date string) store.HistoryRecord {
	t.Helper()

	for _, record := range history {
		if record.Date == date {
			return record
		}
	}
	t.Fatalf("date %s not found in history: %+v", date, history)
	return store.HistoryRecord{}
}

type testClock struct {
	now time.Time
}

func newTestApp(t *testing.T, now time.Time) (*App, *testClock) {
	t.Helper()

	dir := t.TempDir()
	layout := paths.Layout{
		Home:       dir,
		ConfigHome: filepath.Join(dir, ".config"),
		DataHome:   filepath.Join(dir, ".local", "share"),
		StateHome:  filepath.Join(dir, ".local", "state"),
	}
	if err := layout.Ensure(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}

	cfg := config.Default()
	cfg.Schedule.Timezone = "America/Los_Angeles"
	if err := config.Save(layout.ConfigFile(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	sqliteStore, err := store.Open(layout.HistoryDB())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	clock := &testClock{now: now}
	return &App{
		Paths:   layout,
		Runtime: runtime.New(layout.RuntimeFile(), layout.RuntimeLockFile()),
		Store:   sqliteStore,
		Now: func() time.Time {
			return clock.now
		},
	}, clock
}
