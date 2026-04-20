package store

import (
	"context"
	"time"
)

type Event struct {
	SessionDate string
	Timestamp   time.Time
	Command     string
	MatchedRule string
	Action      string
	Outcome     string
	Tier        string
	Shell       string
}

type SessionRecord struct {
	Date              string
	BedtimeConfigured string
	WakeConfigured    string
	FirstBlockedAt    *time.Time
	LastCommandAt     *time.Time
	SnoozesUsed       int
	Skipped           bool
	ForcedActive      bool
}

type HistoryRecord struct {
	Date         string
	Status       string
	SnoozesUsed  int
	BlockedCount int
	Overrides    int
	LastCommand  *time.Time
}

type Stats struct {
	RespectedNights      int
	SnoozedNights        int
	OverriddenNights     int
	CurrentStreak        int
	LongestStreak        int
	MostAttemptedCommand string
	MostAttemptedCount   int
}

type Store interface {
	RecordEvent(context.Context, Event) error
	UpsertSession(context.Context, SessionRecord) error
	History(context.Context, string) ([]HistoryRecord, error)
	Stats(context.Context, string) (Stats, error)
	Purge(context.Context, string) error
	Close() error
}
