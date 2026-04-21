package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteUsesExplicitDateWindows(t *testing.T) {
	t.Parallel()

	sqliteStore, err := Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	now := time.Date(2026, 4, 20, 1, 30, 0, 0, time.UTC)
	lastCommand := now

	for _, date := range []string{"2026-04-19", "2026-04-20"} {
		if err := sqliteStore.UpsertSession(context.Background(), SessionRecord{
			Date:              date,
			BedtimeConfigured: "23:00",
			WakeConfigured:    "07:00",
			LastCommandAt:     &lastCommand,
		}); err != nil {
			t.Fatalf("upsert session %s: %v", date, err)
		}
	}
	if err := sqliteStore.RecordEvent(context.Background(), Event{
		SessionDate: "2026-04-20",
		Timestamp:   now,
		Command:     "claude",
		Action:      "block",
		Outcome:     "blocked",
		Tier:        "curfew",
	}); err != nil {
		t.Fatalf("record event: %v", err)
	}

	history, err := sqliteStore.History(context.Background(), "2026-04-20")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 1 || history[0].Date != "2026-04-20" {
		t.Fatalf("history = %+v, want only 2026-04-20", history)
	}

	stats, err := sqliteStore.Stats(context.Background(), "2026-04-20")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalNights != 1 {
		t.Fatalf("total nights = %d, want 1", stats.TotalNights)
	}
	if stats.AdherentNights != 1 {
		t.Fatalf("adherent nights = %d, want 1", stats.AdherentNights)
	}
	if stats.MostAttemptedCommand != "claude" {
		t.Fatalf("most attempted command = %q, want claude", stats.MostAttemptedCommand)
	}
	if len(stats.TopCommands) != 1 || stats.TopCommands[0].Command != "claude" {
		t.Fatalf("top commands = %+v, want claude", stats.TopCommands)
	}

	details, err := sqliteStore.SessionDetails(context.Background(), "2026-04-20")
	if err != nil {
		t.Fatalf("session details: %v", err)
	}
	if !details.Found {
		t.Fatal("expected session details to be found")
	}
	if details.Session.BlockedCount != 1 || len(details.Events) != 1 {
		t.Fatalf("details = %+v, want one blocked event", details)
	}

	if err := sqliteStore.Purge(context.Background(), "2026-04-20"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	history, err = sqliteStore.History(context.Background(), "2026-01-01")
	if err != nil {
		t.Fatalf("history after purge: %v", err)
	}
	if len(history) != 1 || history[0].Date != "2026-04-20" {
		t.Fatalf("history after purge = %+v, want only 2026-04-20", history)
	}
}

func TestSQLiteTopCommandsAreRankedDeterministically(t *testing.T) {
	t.Parallel()

	sqliteStore, err := Open(filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = sqliteStore.Close()
	})

	now := time.Date(2026, 4, 20, 1, 30, 0, 0, time.UTC)
	for _, event := range []Event{
		{SessionDate: "2026-04-20", Timestamp: now, Command: "claude", Action: "block", Outcome: "blocked"},
		{SessionDate: "2026-04-20", Timestamp: now.Add(time.Second), Command: "aider", Action: "block", Outcome: "blocked"},
		{SessionDate: "2026-04-20", Timestamp: now.Add(2 * time.Second), Command: "claude", Action: "warn", Outcome: "allowed"},
		{SessionDate: "2026-04-20", Timestamp: now.Add(3 * time.Second), Command: "aider", Action: "warn", Outcome: "allowed"},
		{SessionDate: "2026-04-20", Timestamp: now.Add(4 * time.Second), Command: "cursor-agent", Action: "delay", Outcome: "allowed"},
	} {
		if err := sqliteStore.RecordEvent(context.Background(), event); err != nil {
			t.Fatalf("record event %+v: %v", event, err)
		}
	}

	stats, err := sqliteStore.TopCommands(context.Background(), "2026-04-20", 5)
	if err != nil {
		t.Fatalf("top commands: %v", err)
	}
	if len(stats) != 3 {
		t.Fatalf("top commands len = %d, want 3", len(stats))
	}
	if stats[0].Command != "aider" || stats[0].Count != 2 {
		t.Fatalf("top command[0] = %+v, want aider(2)", stats[0])
	}
	if stats[1].Command != "claude" || stats[1].Count != 2 {
		t.Fatalf("top command[1] = %+v, want claude(2)", stats[1])
	}
	if stats[2].Command != "cursor-agent" || stats[2].Count != 1 {
		t.Fatalf("top command[2] = %+v, want cursor-agent(1)", stats[2])
	}
}
