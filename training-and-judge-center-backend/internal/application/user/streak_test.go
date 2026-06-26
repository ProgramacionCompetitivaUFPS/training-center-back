package user

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}

func TestComputeStreak_NoDates(t *testing.T) {
	cur, max := computeStreak(nil, day("2026-06-25"))
	if cur != 0 || max != 0 {
		t.Errorf("want (0,0), got (%d,%d)", cur, max)
	}
}

func TestComputeStreak_SingleDayToday(t *testing.T) {
	now := day("2026-06-25")
	cur, max := computeStreak([]time.Time{day("2026-06-25")}, now)
	if cur != 1 || max != 1 {
		t.Errorf("want (1,1), got (%d,%d)", cur, max)
	}
}

func TestComputeStreak_SingleDayYesterday(t *testing.T) {
	now := day("2026-06-25")
	cur, max := computeStreak([]time.Time{day("2026-06-24")}, now)
	if cur != 1 || max != 1 {
		t.Errorf("want (1,1), got (%d,%d)", cur, max)
	}
}

func TestComputeStreak_SingleDayTwoDaysAgo_CurrentZero(t *testing.T) {
	now := day("2026-06-25")
	cur, max := computeStreak([]time.Time{day("2026-06-23")}, now)
	if cur != 0 || max != 1 {
		t.Errorf("want (0,1), got (%d,%d)", cur, max)
	}
}

func TestComputeStreak_ThreeConsecutiveDaysEndingToday(t *testing.T) {
	now := day("2026-06-25")
	dates := []time.Time{day("2026-06-23"), day("2026-06-24"), day("2026-06-25")}
	cur, max := computeStreak(dates, now)
	if cur != 3 || max != 3 {
		t.Errorf("want (3,3), got (%d,%d)", cur, max)
	}
}

func TestComputeStreak_GapInMiddle_CurrentFromLastRun(t *testing.T) {
	// Dates: 01, 02, 03, [gap], 10, 11 — current run is 2 (10+11=yesterday+today)
	now := day("2026-06-11")
	dates := []time.Time{
		day("2026-06-01"), day("2026-06-02"), day("2026-06-03"),
		day("2026-06-10"), day("2026-06-11"),
	}
	cur, max := computeStreak(dates, now)
	if cur != 2 {
		t.Errorf("want current=2, got %d", cur)
	}
	if max != 3 {
		t.Errorf("want maximum=3, got %d", max)
	}
}

func TestComputeStreak_LongerHistoricalRun(t *testing.T) {
	// Best run: 01-05 (5 days). Current: only yesterday.
	now := day("2026-06-10")
	dates := []time.Time{
		day("2026-06-01"), day("2026-06-02"), day("2026-06-03"),
		day("2026-06-04"), day("2026-06-05"),
		day("2026-06-09"),
	}
	cur, max := computeStreak(dates, now)
	if cur != 1 {
		t.Errorf("want current=1, got %d", cur)
	}
	if max != 5 {
		t.Errorf("want maximum=5, got %d", max)
	}
}

func TestComputeStreak_BrokenCurrentStreak(t *testing.T) {
	// Last submission was 3 days ago — current streak is 0.
	now := day("2026-06-25")
	dates := []time.Time{
		day("2026-06-20"), day("2026-06-21"), day("2026-06-22"),
	}
	cur, max := computeStreak(dates, now)
	if cur != 0 {
		t.Errorf("want current=0, got %d", cur)
	}
	if max != 3 {
		t.Errorf("want maximum=3, got %d", max)
	}
}
