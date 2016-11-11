package util

import "time"

var tz42USA *time.Location

func TZ42USA() *time.Location {
	if tz42USA != nil {
		return tz42USA
	}
	t, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.UTC
	}
	tz42USA = t
	return tz42USA
}
