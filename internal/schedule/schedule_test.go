package schedule

import (
	"testing"
	"time"

	"github.com/rajjoshi/curfew/internal/config"
)

func TestEvaluateOvernightAndOverrides(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Schedule.Timezone = "America/Los_Angeles"

	tests := []struct {
		name            string
		now             string
		wantTier        Tier
		wantSessionDate string
		wantWake        string
		wantBedtime     string
		wantNextDate    string
	}{
		{
			name:            "warning window before default bedtime",
			now:             "2026-04-23T22:45:00-07:00",
			wantTier:        TierWarning,
			wantSessionDate: "2026-04-23",
			wantWake:        "2026-04-24T07:00:00-07:00",
			wantBedtime:     "2026-04-23T23:00:00-07:00",
			wantNextDate:    "2026-04-23",
		},
		{
			name:            "curfew after bedtime",
			now:             "2026-04-23T23:30:00-07:00",
			wantTier:        TierCurfew,
			wantSessionDate: "2026-04-23",
			wantWake:        "2026-04-24T07:00:00-07:00",
			wantBedtime:     "2026-04-23T23:00:00-07:00",
			wantNextDate:    "2026-04-24",
		},
		{
			name:            "hard stop after midnight",
			now:             "2026-04-24T01:30:00-07:00",
			wantTier:        TierHardStop,
			wantSessionDate: "2026-04-23",
			wantWake:        "2026-04-24T07:00:00-07:00",
			wantBedtime:     "2026-04-23T23:00:00-07:00",
			wantNextDate:    "2026-04-24",
		},
		{
			name:         "normal after wake uses next session",
			now:          "2026-04-24T10:00:00-07:00",
			wantTier:     TierNormal,
			wantNextDate: "2026-04-24",
			wantWake:     "",
			wantBedtime:  "",
		},
		{
			name:            "friday override applies to friday night midnight bedtime",
			now:             "2026-04-24T23:45:00-07:00",
			wantTier:        TierWarning,
			wantSessionDate: "2026-04-24",
			wantWake:        "2026-04-25T09:00:00-07:00",
			wantBedtime:     "2026-04-25T00:00:00-07:00",
			wantNextDate:    "2026-04-24",
		},
		{
			name:            "friday override curfew continues after midnight",
			now:             "2026-04-25T00:30:00-07:00",
			wantTier:        TierCurfew,
			wantSessionDate: "2026-04-24",
			wantWake:        "2026-04-25T09:00:00-07:00",
			wantBedtime:     "2026-04-25T00:00:00-07:00",
			wantNextDate:    "2026-04-25",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			now, err := time.Parse(time.RFC3339, test.now)
			if err != nil {
				t.Fatalf("parse time: %v", err)
			}
			eval, err := Evaluate(now, cfg)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if eval.Tier != test.wantTier {
				t.Fatalf("tier = %q, want %q", eval.Tier, test.wantTier)
			}
			if eval.Next.Date != test.wantNextDate {
				t.Fatalf("next date = %q, want %q", eval.Next.Date, test.wantNextDate)
			}
			if test.wantSessionDate == "" {
				if eval.Current != nil {
					t.Fatalf("current session = %v, want nil", eval.Current)
				}
				return
			}
			if eval.Current == nil {
				t.Fatal("current session is nil")
			}
			if eval.Current.Date != test.wantSessionDate {
				t.Fatalf("session date = %q, want %q", eval.Current.Date, test.wantSessionDate)
			}
			if got := eval.Current.Wake.Format(time.RFC3339); got != test.wantWake {
				t.Fatalf("wake = %q, want %q", got, test.wantWake)
			}
			if got := eval.Current.Bedtime.Format(time.RFC3339); got != test.wantBedtime {
				t.Fatalf("bedtime = %q, want %q", got, test.wantBedtime)
			}
		})
	}
}

func TestEvaluateOnDSTBoundary(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Schedule.Timezone = "America/Los_Angeles"

	now, err := time.Parse(time.RFC3339, "2026-03-08T01:30:00-08:00")
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	eval, err := Evaluate(now, cfg)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if eval.Current == nil {
		t.Fatal("expected a current session on the DST boundary")
	}
	if eval.Current.Wake.Before(eval.Now) {
		t.Fatalf("wake %s should be after now %s", eval.Current.Wake, eval.Now)
	}
}
