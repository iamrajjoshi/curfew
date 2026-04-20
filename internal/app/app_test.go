package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rajjoshi/curfew/internal/config"
	"github.com/rajjoshi/curfew/internal/paths"
	"github.com/rajjoshi/curfew/internal/runtime"
	"github.com/rajjoshi/curfew/internal/store"
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
	if len(history) != 1 {
		t.Fatalf("history records = %d, want 1", len(history))
	}
	if history[0].Status != "overrode" {
		t.Fatalf("history status = %q, want overrode", history[0].Status)
	}

	stats, err := application.Stats(context.Background(), 7)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.OverriddenNights != 1 {
		t.Fatalf("overridden nights = %d, want 1", stats.OverriddenNights)
	}
	if stats.MostAttemptedCommand != "claude" {
		t.Fatalf("most attempted command = %q, want claude", stats.MostAttemptedCommand)
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
	now := time.Date(2026, 4, 24, 23, 30, 0, 0, location)
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
	if stopSession.Date != "2026-04-24" {
		t.Fatalf("stop session = %q, want 2026-04-24", stopSession.Date)
	}

	status, err := application.EvaluateStatus()
	if err != nil {
		t.Fatalf("evaluate status after stop: %v", err)
	}
	if !status.Disabled {
		t.Fatal("expected stop tonight to disable the current session")
	}

	clock.now = time.Date(2026, 4, 25, 23, 45, 0, 0, location)
	skipSession, err := application.SkipTonight(
		context.Background(),
		"Skip tonight's curfew?",
		strings.NewReader("i am choosing to break my own rule\n"),
		&strings.Builder{},
	)
	if err != nil {
		t.Fatalf("skip tonight: %v", err)
	}
	if skipSession.Date != "2026-04-25" {
		t.Fatalf("skip session = %q, want 2026-04-25", skipSession.Date)
	}

	history, err := application.History(context.Background(), 7)
	if err != nil {
		t.Fatalf("history after skip: %v", err)
	}
	if len(history) == 0 || history[0].Status != "overrode" {
		t.Fatalf("expected skip to register as an override, got %+v", history)
	}
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
