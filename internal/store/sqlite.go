package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLite struct {
	db *sql.DB
}

func Open(path string) (*SQLite, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	store := &SQLite{db: db}
	if err := store.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *SQLite) init() error {
	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA busy_timeout = 5000;`,
		`CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY,
			session_date TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			command TEXT NOT NULL,
			matched_rule TEXT,
			action TEXT NOT NULL,
			outcome TEXT NOT NULL,
			tier TEXT,
			shell TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			date TEXT PRIMARY KEY,
			bedtime_configured TEXT,
			wake_configured TEXT,
			first_blocked_at TEXT,
			last_command_at TEXT,
			snoozes_used INTEGER DEFAULT 0,
			skipped INTEGER DEFAULT 0,
			forced_active INTEGER DEFAULT 0
		);`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLite) RecordEvent(ctx context.Context, event Event) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO events (session_date, timestamp, command, matched_rule, action, outcome, tier, shell)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.SessionDate,
		event.Timestamp.Unix(),
		event.Command,
		nullIfEmpty(event.MatchedRule),
		event.Action,
		event.Outcome,
		nullIfEmpty(event.Tier),
		nullIfEmpty(event.Shell),
	)
	return err
}

func (s *SQLite) UpsertSession(ctx context.Context, record SessionRecord) error {
	var firstBlocked any
	if record.FirstBlockedAt != nil {
		firstBlocked = record.FirstBlockedAt.Format(time.RFC3339)
	}
	var lastCommand any
	if record.LastCommandAt != nil {
		lastCommand = record.LastCommandAt.Format(time.RFC3339)
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (date, bedtime_configured, wake_configured, first_blocked_at, last_command_at, snoozes_used, skipped, forced_active)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(date) DO UPDATE SET
		   bedtime_configured = excluded.bedtime_configured,
		   wake_configured = excluded.wake_configured,
		   first_blocked_at = COALESCE(sessions.first_blocked_at, excluded.first_blocked_at),
		   last_command_at = COALESCE(excluded.last_command_at, sessions.last_command_at),
		   snoozes_used = MAX(sessions.snoozes_used, excluded.snoozes_used),
		   skipped = MAX(sessions.skipped, excluded.skipped),
		   forced_active = excluded.forced_active`,
		record.Date,
		record.BedtimeConfigured,
		record.WakeConfigured,
		firstBlocked,
		lastCommand,
		record.SnoozesUsed,
		boolToInt(record.Skipped),
		boolToInt(record.ForcedActive),
	)
	return err
}

func (s *SQLite) History(ctx context.Context, days int) ([]HistoryRecord, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`WITH event_rollup AS (
			SELECT
			  session_date,
			  SUM(CASE WHEN outcome = 'blocked' THEN 1 ELSE 0 END) AS blocked_count,
			  SUM(CASE WHEN outcome = 'overridden' THEN 1 ELSE 0 END) AS overridden_count
			FROM events
			WHERE session_date >= date('now', printf('-%d day', ?))
			GROUP BY session_date
		)
		SELECT
		  s.date,
		  s.snoozes_used,
		  COALESCE(e.blocked_count, 0),
		  COALESCE(e.overridden_count, 0),
		  s.last_command_at,
		  s.skipped
		FROM sessions s
		LEFT JOIN event_rollup e ON e.session_date = s.date
		WHERE s.date >= date('now', printf('-%d day', ?))
		ORDER BY s.date DESC`,
		days, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []HistoryRecord
	for rows.Next() {
		var record HistoryRecord
		var lastCommand sql.NullString
		var skipped int
		if err := rows.Scan(&record.Date, &record.SnoozesUsed, &record.BlockedCount, &record.Overrides, &lastCommand, &skipped); err != nil {
			return nil, err
		}
		if lastCommand.Valid {
			parsed, err := time.Parse(time.RFC3339, lastCommand.String)
			if err == nil {
				record.LastCommand = &parsed
			}
		}
		record.Status = summarizeStatus(skipped == 1, record.SnoozesUsed, record.Overrides)
		history = append(history, record)
	}
	return history, rows.Err()
}

func (s *SQLite) Stats(ctx context.Context, days int) (Stats, error) {
	history, err := s.History(ctx, days)
	if err != nil {
		return Stats{}, err
	}

	var stats Stats
	for _, record := range history {
		switch record.Status {
		case "respected":
			stats.RespectedNights++
		case "snoozed":
			stats.SnoozedNights++
		case "overrode":
			stats.OverriddenNights++
		}
	}

	streak := 0
	longest := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Status == "respected" || history[i].Status == "snoozed" {
			streak++
			if streak > longest {
				longest = streak
			}
			continue
		}
		streak = 0
	}
	for _, record := range history {
		if record.Status == "respected" || record.Status == "snoozed" {
			stats.CurrentStreak++
			continue
		}
		break
	}
	stats.LongestStreak = longest

	row := s.db.QueryRowContext(
		ctx,
		`SELECT command, COUNT(*)
		 FROM events
		 WHERE session_date >= date('now', printf('-%d day', ?))
		   AND action IN ('block', 'warn', 'delay')
		 GROUP BY command
		 ORDER BY COUNT(*) DESC, command ASC
		 LIMIT 1`,
		days,
	)
	_ = row.Scan(&stats.MostAttemptedCommand, &stats.MostAttemptedCount)

	return stats, nil
}

func (s *SQLite) Purge(ctx context.Context, retainDays int) error {
	cutoff := fmt.Sprintf("-%d day", retainDays)
	if _, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE session_date < date('now', ?)`, cutoff); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE date < date('now', ?)`, cutoff)
	return err
}

func (s *SQLite) Close() error {
	return s.db.Close()
}

func summarizeStatus(skipped bool, snoozes int, overrides int) string {
	if skipped || overrides > 0 {
		return "overrode"
	}
	if snoozes > 0 {
		return "snoozed"
	}
	return "respected"
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
