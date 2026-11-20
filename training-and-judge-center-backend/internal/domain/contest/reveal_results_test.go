package contest_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var (
	revealStart = time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
	revealEnd   = revealStart.Add(5 * time.Hour)
)

func contestWithFreeze(t *testing.T, freezeMinutes int, revealedAt *time.Time) *contest.Contest {
	t.Helper()
	return contest.RestoreContest(
		"id",
		mustName(t, "Test"),
		nil,
		revealStart, revealEnd,
		mustPenalty(t, 20),
		freezeMinutes,
		false, false, false, false,
		revealedAt,
		nil,
		shared.RestoreGroupID("g"),
		shared.RestoreUserID("o"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		nil,
		revealStart.Add(-time.Hour),
		nil,
	)
}

func TestFreezeTime(t *testing.T) {
	freezeAt := revealEnd.Add(-60 * time.Minute)
	revealed := revealEnd.Add(time.Hour)

	tests := []struct {
		name          string
		freezeMinutes int
		revealedAt    *time.Time
		now           time.Time
		wantFrozen    bool
	}{
		{"before the freeze", 60, nil, revealEnd.Add(-2 * time.Hour), false},
		{"inside the freeze", 60, nil, revealEnd.Add(-10 * time.Minute), true},
		{"finished and not revealed", 60, nil, revealEnd.Add(time.Minute), true},
		{"finished and revealed", 60, &revealed, revealEnd.Add(2 * time.Hour), false},
		{"no freeze while running", 0, nil, revealEnd.Add(-time.Minute), false},
		{"no freeze once finished", 0, nil, revealEnd.Add(time.Minute), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contestWithFreeze(t, tt.freezeMinutes, tt.revealedAt).FreezeTime(tt.now)
			if (got != nil) != tt.wantFrozen {
				t.Fatalf("FreezeTime frozen = %v, want %v", got != nil, tt.wantFrozen)
			}
			if got != nil && !got.Equal(freezeAt) {
				t.Errorf("FreezeTime = %v, want %v", got, freezeAt)
			}
		})
	}
}

func TestRevealResults_FinishedContestWithFreeze_Reveals(t *testing.T) {
	c := contestWithFreeze(t, 60, nil)
	now := revealEnd.Add(time.Minute)

	if err := c.RevealResults(now); err != nil {
		t.Fatalf("RevealResults: unexpected error %v", err)
	}
	if c.ResultsRevealedAt() == nil || !c.ResultsRevealedAt().Equal(now) {
		t.Errorf("ResultsRevealedAt = %v, want %v", c.ResultsRevealedAt(), now)
	}
	if c.UpdatedAt() == nil || !c.UpdatedAt().Equal(now) {
		t.Errorf("UpdatedAt = %v, want %v", c.UpdatedAt(), now)
	}
	if c.FreezeTime(now) != nil {
		t.Error("FreezeTime must be nil after revealing")
	}
}

func TestRevealResults_AlreadyRevealed_KeepsFirstInstant(t *testing.T) {
	first := revealEnd.Add(time.Minute)
	c := contestWithFreeze(t, 60, &first)

	if err := c.RevealResults(first.Add(time.Hour)); err != nil {
		t.Fatalf("RevealResults: unexpected error %v", err)
	}
	if !c.ResultsRevealedAt().Equal(first) {
		t.Errorf("ResultsRevealedAt = %v, want the first instant %v", c.ResultsRevealedAt(), first)
	}
}

func TestRevealResults_Rejections(t *testing.T) {
	tests := []struct {
		name          string
		freezeMinutes int
		now           time.Time
		wantCode      string
	}{
		{"contest without freeze", 0, revealEnd.Add(time.Minute), contest.ErrCodeContestHasNoFreeze},
		{"contest still running", 60, revealEnd.Add(-10 * time.Minute), contest.ErrCodeContestNotFinished},
		{"contest not started", 60, revealStart.Add(-time.Minute), contest.ErrCodeContestNotFinished},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := contestWithFreeze(t, tt.freezeMinutes, nil)

			err := c.RevealResults(tt.now)

			appErr, ok := err.(*apperror.AppError)
			if !ok || appErr.Code != tt.wantCode {
				t.Fatalf("RevealResults error = %v, want code %s", err, tt.wantCode)
			}
			if c.ResultsRevealedAt() != nil {
				t.Error("a rejected reveal must not set ResultsRevealedAt")
			}
		})
	}
}

func contestWithAutoReveal(t *testing.T, freezeMinutes int, autoRevealMinutes int) *contest.Contest {
	t.Helper()
	c := contestWithFreeze(t, freezeMinutes, nil)
	if err := c.SetAutoRevealMinutes(&autoRevealMinutes, revealStart); err != nil {
		t.Fatalf("SetAutoRevealMinutes: %v", err)
	}
	return c
}

func TestFreezeTime_AutoReveal(t *testing.T) {
	tests := []struct {
		name       string
		now        time.Time
		wantFrozen bool
	}{
		{"still running", revealEnd.Add(-10 * time.Minute), true},
		{"just finished, inside the delay", revealEnd.Add(10 * time.Minute), true},
		{"exactly at the auto reveal time", revealEnd.Add(30 * time.Minute), false},
		{"after the auto reveal time", revealEnd.Add(time.Hour), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contestWithAutoReveal(t, 60, 30).FreezeTime(tt.now)
			if (got != nil) != tt.wantFrozen {
				t.Fatalf("FreezeTime frozen = %v, want %v", got != nil, tt.wantFrozen)
			}
		})
	}
}

func TestFreezeTime_AutoRevealZeroOpensAtTheEnd(t *testing.T) {
	c := contestWithAutoReveal(t, 60, 0)

	if c.FreezeTime(revealEnd.Add(-time.Minute)) == nil {
		t.Error("the freeze must still apply while the contest runs")
	}
	if c.FreezeTime(revealEnd) != nil {
		t.Error("with autoRevealMinutes = 0 the results open as soon as the contest ends")
	}
}

func TestAutoRevealAt(t *testing.T) {
	if at := contestWithAutoReveal(t, 60, 30).AutoRevealAt(); at == nil || !at.Equal(revealEnd.Add(30*time.Minute)) {
		t.Errorf("AutoRevealAt = %v, want endTime + 30m", at)
	}
	if at := contestWithFreeze(t, 60, nil).AutoRevealAt(); at != nil {
		t.Errorf("AutoRevealAt = %v, want nil in manual mode", at)
	}
	if at := contestWithAutoReveal(t, 0, 30).AutoRevealAt(); at != nil {
		t.Errorf("AutoRevealAt = %v, want nil when the contest has no freeze", at)
	}
}

func TestRevealResults_AfterAutoRevealTime_ChangesNothing(t *testing.T) {
	c := contestWithAutoReveal(t, 60, 30)

	if err := c.RevealResults(revealEnd.Add(time.Hour)); err != nil {
		t.Fatalf("RevealResults: unexpected error %v", err)
	}
	if c.ResultsRevealedAt() != nil {
		t.Error("results already open by themselves: no reveal instant must be recorded")
	}
}

func TestRevealResults_BeforeAutoRevealTime_RevealsEarly(t *testing.T) {
	c := contestWithAutoReveal(t, 60, 30)
	now := revealEnd.Add(5 * time.Minute)

	if err := c.RevealResults(now); err != nil {
		t.Fatalf("RevealResults: unexpected error %v", err)
	}
	if c.ResultsRevealedAt() == nil || !c.ResultsRevealedAt().Equal(now) {
		t.Errorf("ResultsRevealedAt = %v, want %v", c.ResultsRevealedAt(), now)
	}
}

func TestSetAutoRevealMinutes(t *testing.T) {
	c := contestWithFreeze(t, 60, nil)
	negative := -1

	err := c.SetAutoRevealMinutes(&negative, revealStart)
	if appErr, ok := err.(*apperror.AppError); !ok || appErr.Code != apperror.ErrCodeValidationError {
		t.Fatalf("negative minutes: error = %v, want a validation error", err)
	}
	if c.AutoRevealMinutes() != nil {
		t.Error("a rejected value must not be stored")
	}

	minutes := 15
	if err := c.SetAutoRevealMinutes(&minutes, revealStart); err != nil || *c.AutoRevealMinutes() != 15 {
		t.Fatalf("set 15: err=%v value=%v", err, c.AutoRevealMinutes())
	}
	minutes = 99
	if *c.AutoRevealMinutes() != 15 {
		t.Error("the contest must keep its own copy of the value")
	}

	if err := c.SetAutoRevealMinutes(nil, revealStart); err != nil || c.AutoRevealMinutes() != nil {
		t.Fatalf("back to manual: err=%v value=%v", err, c.AutoRevealMinutes())
	}
}
