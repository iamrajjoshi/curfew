package config

import "time"

func ParseClockForSchedule(value string) (time.Duration, error) {
	return parseClock(value)
}
