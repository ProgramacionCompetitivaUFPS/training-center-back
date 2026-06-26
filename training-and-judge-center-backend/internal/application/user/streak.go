package user

import "time"

// computeStreak returns (current, maximum) streak from a sorted-ascending slice
// of distinct UTC-midnight dates on which the user made at least one submission.
//
// current: consecutive days ending today or yesterday (0 if last submission
//
//	was 2+ days ago).
//
// maximum: longest consecutive run in history.
func computeStreak(dates []time.Time, now time.Time) (current, maximum int) {
	if len(dates) == 0 {
		return 0, 0
	}

	today := now.UTC().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	day := 24 * time.Hour

	// Maximum streak — single pass over ascending dates.
	run := 1
	max := 1
	for i := 1; i < len(dates); i++ {
		if dates[i].Sub(dates[i-1]) == day {
			run++
			if run > max {
				max = run
			}
		} else {
			run = 1
		}
	}
	maximum = max

	// Current streak — only valid if the most recent submission was today or yesterday.
	last := dates[len(dates)-1]
	if !last.Equal(today) && !last.Equal(yesterday) {
		return 0, maximum
	}

	cur := 1
	for i := len(dates) - 2; i >= 0; i-- {
		if dates[i+1].Sub(dates[i]) == day {
			cur++
		} else {
			break
		}
	}
	return cur, maximum
}
