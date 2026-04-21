package store

import (
	"context"
	"time"
)

type Event struct {
	SessionDate string    `json:"session_date"`
	Timestamp   time.Time `json:"timestamp"`
	Command     string    `json:"command"`
	MatchedRule string    `json:"matched_rule,omitempty"`
	Action      string    `json:"action"`
	Outcome     string    `json:"outcome"`
	Tier        string    `json:"tier,omitempty"`
	Shell       string    `json:"shell,omitempty"`
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
	Date         string     `json:"date"`
	Status       string     `json:"status"`
	SnoozesUsed  int        `json:"snoozes_used"`
	BlockedCount int        `json:"blocked_count"`
	Overrides    int        `json:"overrides"`
	LastCommand  *time.Time `json:"last_command,omitempty"`
}

type Stats struct {
	TotalNights          int           `json:"total_nights"`
	RespectedNights      int           `json:"respected_nights"`
	SnoozedNights        int           `json:"snoozed_nights"`
	OverriddenNights     int           `json:"overridden_nights"`
	AdherentNights       int           `json:"adherent_nights"`
	AdherenceRate        float64       `json:"adherence_rate"`
	CurrentStreak        int           `json:"current_streak"`
	LongestStreak        int           `json:"longest_streak"`
	MostAttemptedCommand string        `json:"most_attempted_command,omitempty"`
	MostAttemptedCount   int           `json:"most_attempted_count,omitempty"`
	TopCommands          []CommandStat `json:"top_commands,omitempty"`
}

type CommandStat struct {
	Command string `json:"command"`
	Count   int    `json:"count"`
}

type SessionDetail struct {
	Date              string     `json:"date"`
	Status            string     `json:"status"`
	BedtimeConfigured string     `json:"bedtime_configured"`
	WakeConfigured    string     `json:"wake_configured"`
	FirstBlockedAt    *time.Time `json:"first_blocked_at,omitempty"`
	LastCommandAt     *time.Time `json:"last_command_at,omitempty"`
	SnoozesUsed       int        `json:"snoozes_used"`
	Skipped           bool       `json:"skipped"`
	ForcedActive      bool       `json:"forced_active"`
	BlockedCount      int        `json:"blocked_count"`
	Overrides         int        `json:"overrides"`
}

type SessionDetails struct {
	Found   bool          `json:"found"`
	Session SessionDetail `json:"session"`
	Events  []Event       `json:"events"`
}

type Store interface {
	RecordEvent(context.Context, Event) error
	UpsertSession(context.Context, SessionRecord) error
	History(context.Context, string) ([]HistoryRecord, error)
	SessionDetails(context.Context, string, *time.Location) (SessionDetails, error)
	TopCommands(context.Context, string, int) ([]CommandStat, error)
	Stats(context.Context, string) (Stats, error)
	Purge(context.Context, string) error
	Close() error
}
