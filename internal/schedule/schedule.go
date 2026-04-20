package schedule

import (
	"fmt"
	"time"

	"github.com/rajjoshi/curfew/internal/config"
)

type Tier string

const (
	TierNormal   Tier = "normal"
	TierWarning  Tier = "warning"
	TierCurfew   Tier = "curfew"
	TierHardStop Tier = "hard_stop"
)

type Session struct {
	Date         string        `json:"date"`
	LocationName string        `json:"location_name"`
	Bedtime      time.Time     `json:"bedtime"`
	Wake         time.Time     `json:"wake"`
	WarningStart time.Time     `json:"warning_start"`
	HardStop     time.Time     `json:"hard_stop"`
	BedtimeText  string        `json:"bedtime_text"`
	WakeText     string        `json:"wake_text"`
	HardStopText string        `json:"hard_stop_text"`
	Warning      time.Duration `json:"warning"`
}

type Evaluation struct {
	Now         time.Time `json:"now"`
	Location    *time.Location
	Current     *Session `json:"current,omitempty"`
	Next        Session  `json:"next"`
	Tier        Tier     `json:"tier"`
	TimeUntil   time.Duration
	InQuietTime bool `json:"in_quiet_time"`
}

func ResolveLocation(tz string) (*time.Location, error) {
	if tz == "" || tz == "auto" {
		return time.Local, nil
	}
	return time.LoadLocation(tz)
}

func Evaluate(now time.Time, cfg config.Config) (Evaluation, error) {
	location, err := ResolveLocation(cfg.Schedule.Timezone)
	if err != nil {
		return Evaluation{}, err
	}
	now = now.In(location)

	var current *Session
	for _, candidate := range []time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location),
		time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, location),
	} {
		session, err := SessionForDate(candidate, cfg, location)
		if err != nil {
			return Evaluation{}, err
		}
		if !now.Before(session.WarningStart) && now.Before(session.Wake) {
			current = &session
			break
		}
	}

	next, err := NextSession(now, cfg, location)
	if err != nil {
		return Evaluation{}, err
	}

	eval := Evaluation{
		Now:      now,
		Location: location,
		Current:  current,
		Next:     next,
		Tier:     TierNormal,
	}
	if current == nil {
		eval.TimeUntil = next.Bedtime.Sub(now)
		return eval, nil
	}

	eval.InQuietTime = !now.Before(current.Bedtime)
	eval.TimeUntil = current.Wake.Sub(now)

	switch {
	case now.Before(current.Bedtime):
		eval.Tier = TierWarning
	case !now.Before(current.HardStop):
		eval.Tier = TierHardStop
	default:
		eval.Tier = TierCurfew
	}

	return eval, nil
}

func NextSession(now time.Time, cfg config.Config, location *time.Location) (Session, error) {
	for offset := 0; offset < 8; offset++ {
		candidateDate := time.Date(now.Year(), now.Month(), now.Day()+offset, 0, 0, 0, 0, location)
		session, err := SessionForDate(candidateDate, cfg, location)
		if err != nil {
			return Session{}, err
		}
		if session.Bedtime.After(now) {
			return session, nil
		}
	}
	return Session{}, fmt.Errorf("could not resolve next session")
}

func SessionForDate(date time.Time, cfg config.Config, location *time.Location) (Session, error) {
	daySchedule := cfg.Schedule.Default
	if override, ok := cfg.Schedule.Overrides[config.WeekdayKey(date.In(location).Weekday())]; ok {
		daySchedule = override
	}

	bedtimeDuration, err := parseClock(daySchedule.Bedtime)
	if err != nil {
		return Session{}, err
	}
	wakeDuration, err := parseClock(daySchedule.Wake)
	if err != nil {
		return Session{}, err
	}
	warningDuration, err := time.ParseDuration(cfg.Grace.WarningWindow)
	if err != nil {
		return Session{}, err
	}
	hardStopDuration, err := parseClock(cfg.Grace.HardStopAfter)
	if err != nil {
		return Session{}, err
	}

	base := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
	bedtime := base.Add(bedtimeDuration)
	if bedtimeDuration < 12*time.Hour {
		bedtime = bedtime.Add(24 * time.Hour)
	}
	wake := base.Add(wakeDuration)
	if bedtimeDuration < 12*time.Hour {
		wake = wake.Add(24 * time.Hour)
	}
	if !wake.After(bedtime) {
		wake = wake.Add(24 * time.Hour)
	}
	hardStop := base.Add(hardStopDuration)
	if hardStopDuration < 12*time.Hour {
		hardStop = hardStop.Add(24 * time.Hour)
	}
	if !hardStop.After(bedtime) {
		hardStop = hardStop.Add(24 * time.Hour)
	}

	return Session{
		Date:         base.Format("2006-01-02"),
		LocationName: location.String(),
		Bedtime:      bedtime,
		Wake:         wake,
		WarningStart: bedtime.Add(-warningDuration),
		HardStop:     hardStop,
		BedtimeText:  daySchedule.Bedtime,
		WakeText:     daySchedule.Wake,
		HardStopText: cfg.Grace.HardStopAfter,
		Warning:      warningDuration,
	}, nil
}

func parseClock(value string) (time.Duration, error) {
	return config.ParseClockForSchedule(value)
}
