package contest_test

import (
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

var (
	fixedNow   = time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	fixedClock = func() time.Time { return fixedNow }
)

func mustName(t *testing.T, s string) contest.ContestName {
	t.Helper()
	n, err := contest.NewContestName(s)
	if err != nil {
		t.Fatalf("mustName: %v", err)
	}
	return n
}

func mustPenalty(t *testing.T, v int) contest.Penalty {
	t.Helper()
	p, err := contest.NewPenalty(v)
	if err != nil {
		t.Fatalf("mustPenalty: %v", err)
	}
	return p
}

func validContest(t *testing.T) (*contest.Contest, error) {
	t.Helper()
	return contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		fixedClock,
	)
}

func TestNewContest_Valid(t *testing.T) {
	c, err := validContest(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID() != "contest-id" {
		t.Errorf("unexpected id: %s", c.ID())
	}
	if c.Penalty().Value() != 20 {
		t.Errorf("unexpected penalty: %d", c.Penalty().Value())
	}
	if c.FreezeMinutes() != 60 {
		t.Errorf("unexpected freezeMinutes: %d", c.FreezeMinutes())
	}
	if c.Locked() {
		t.Error("new contest should not be locked")
	}
	if len(c.Problems()) != 0 {
		t.Error("new contest should have no problems")
	}
}

func TestNewContest_StartTimeInPast(t *testing.T) {
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(-time.Hour), // past
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		fixedClock,
	)
	if err == nil {
		t.Fatal("expected error for past startTime, got nil")
	}
}

func TestNewContest_EndBeforeStart(t *testing.T) {
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(2*time.Hour),
		fixedNow.Add(time.Hour), // before start
		mustPenalty(t, 20),
		60,
		false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		fixedClock,
	)
	if err == nil {
		t.Fatal("expected error for endTime before startTime, got nil")
	}
}

func TestNewContest_EndEqualStart(t *testing.T) {
	start := fixedNow.Add(time.Hour)
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		start,
		start, // equal
		mustPenalty(t, 20),
		60,
		false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		fixedClock,
	)
	if err == nil {
		t.Fatal("expected error for endTime equal to startTime, got nil")
	}
}

func TestNewContest_DescriptionTooLong(t *testing.T) {
	long := strings.Repeat("x", 5001)
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		&long,
		fixedNow.Add(time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		fixedClock,
	)
	if err == nil {
		t.Fatal("expected error for description too long, got nil")
	}
}

func TestNewContest_NegativeFreezeMinutes(t *testing.T) {
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		-1,
		false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		fixedClock,
	)
	if err == nil {
		t.Fatal("expected error for negative freezeMinutes, got nil")
	}
}

func TestContest_Status(t *testing.T) {
	start := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		now    time.Time
		status contest.Status
	}{
		{"before start", start.Add(-time.Minute), contest.StatusScheduled},
		{"at start", start, contest.StatusActive},
		{"during contest", start.Add(time.Hour), contest.StatusActive},
		{"at end", end, contest.StatusActive},
		{"after end", end.Add(time.Second), contest.StatusFinished},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := contest.RestoreContest(
				"id",
				mustName(t, "Test"),
				nil,
				start,
				end,
				mustPenalty(t, 20),
				60,
				false,
				false,
				shared.RestoreGroupID("g"),
				shared.RestoreUserID("o"),
				nil,
				time.Now(),
				nil,
			)
			c.WithClock(func() time.Time { return tc.now })
			if got := c.Status(); got != tc.status {
				t.Errorf("at %v: expected %s, got %s", tc.now, tc.status, got)
			}
		})
	}
}

func TestContest_Duration(t *testing.T) {
	c, _ := validContest(t)
	if got := c.Duration(); got != 180 {
		t.Errorf("expected 180 minutes, got %d", got)
	}
}

func TestContest_AddProblem(t *testing.T) {
	c, _ := validContest(t)

	c.AddProblem("cp-1", "problem-a")
	c.AddProblem("cp-2", "problem-b")
	c.AddProblem("cp-3", "problem-c")

	if len(c.Problems()) != 3 {
		t.Fatalf("expected 3 problems, got %d", len(c.Problems()))
	}
	if c.Problems()[0].Order() != 1 || c.Problems()[1].Order() != 2 || c.Problems()[2].Order() != 3 {
		t.Error("problems not in sequential order")
	}
}

func TestContest_AddProblem_Deduplication(t *testing.T) {
	c, _ := validContest(t)

	c.AddProblem("cp-1", "problem-a")
	c.AddProblem("cp-2", "problem-a") // duplicate

	if len(c.Problems()) != 1 {
		t.Errorf("expected 1 problem after deduplication, got %d", len(c.Problems()))
	}
}

func TestContest_RemoveProblem(t *testing.T) {
	c, _ := validContest(t)

	c.AddProblem("cp-1", "problem-a")
	c.AddProblem("cp-2", "problem-b")
	c.AddProblem("cp-3", "problem-c")

	removed := c.RemoveProblem("problem-b")
	if !removed {
		t.Fatal("expected RemoveProblem to return true")
	}
	if len(c.Problems()) != 2 {
		t.Fatalf("expected 2 problems after removal, got %d", len(c.Problems()))
	}
	// Orders must be resequenced
	if c.Problems()[0].Order() != 1 || c.Problems()[1].Order() != 2 {
		t.Error("orders not resequenced after removal")
	}
}

func TestContest_RemoveProblem_NotFound(t *testing.T) {
	c, _ := validContest(t)
	removed := c.RemoveProblem("non-existent")
	if removed {
		t.Error("expected RemoveProblem to return false for non-existent problem")
	}
}
