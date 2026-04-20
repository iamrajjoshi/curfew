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
	if stats.MostAttemptedCommand != "claude" {
		t.Fatalf("most attempted command = %q, want claude", stats.MostAttemptedCommand)
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
