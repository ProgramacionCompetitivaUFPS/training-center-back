package timeutil

import "time"

const RFC3339UTC = "2006-01-02T15:04:05Z"

func Format(t time.Time) string {
	return t.UTC().Format(RFC3339UTC)
}
