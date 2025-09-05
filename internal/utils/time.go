package utils

import "time"

func RollingWindow(at time.Time, window, drift time.Duration) (time.Time, time.Time) {
	end := at.UTC()
	start := end.Add(-window - drift)
	return start, end
}
