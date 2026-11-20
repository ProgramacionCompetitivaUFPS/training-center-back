package contest_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var fixedNow = time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)

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

func validContest(t *testing.T) *contest.Contest {
	t.Helper()
	c, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	if err != nil {
		t.Fatalf("validContest: %v", err)
	}
	return c
}

func TestNewContest_Valid(t *testing.T) {
	c := validContest(t)
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
		fixedNow.Add(-time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	if err == nil {
		t.Fatal("expected error for past startTime, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != contest.ErrCodeStartTimeInPast {
		t.Errorf("expected code %q, got %q", contest.ErrCodeStartTimeInPast, appErr.Code)
	}
}

func TestNewContest_EndBeforeStart(t *testing.T) {
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(2*time.Hour),
		fixedNow.Add(time.Hour),
		mustPenalty(t, 20),
		60,
		false,
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	if err == nil {
		t.Fatal("expected error for endTime before startTime, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != contest.ErrCodeInvalidTimeRange {
		t.Errorf("expected code %q, got %q", contest.ErrCodeInvalidTimeRange, appErr.Code)
	}
}

func TestNewContest_EndEqualStart(t *testing.T) {
	start := fixedNow.Add(time.Hour)
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		start,
		start,
		mustPenalty(t, 20),
		60,
		false,
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	if err == nil {
		t.Fatal("expected error for endTime equal to startTime, got nil")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != contest.ErrCodeInvalidTimeRange {
		t.Errorf("expected code %q, got %q", contest.ErrCodeInvalidTimeRange, appErr.Code)
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
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	assertValidationField(t, "NewContest(descriptionTooLong)", err, "description")
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
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	assertValidationField(t, "NewContest(negativeFreezeMinutes)", err, "freezeMinutes")
}

func TestNewContest_FreezeMinutesExceedsDuration(t *testing.T) {
	_, err := contest.NewContest(
		"contest-id",
		mustName(t, "Weekly Contest"),
		nil,
		fixedNow.Add(time.Hour),
		fixedNow.Add(2*time.Hour), // duration = 60 min
		mustPenalty(t, 20),
		60, // freeze == duration → invalid
		false,
		false, false,
		shared.RestoreGroupID("group-id"),
		shared.RestoreUserID("owner-id"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		fixedNow,
	)
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != contest.ErrCodeFreezeTooLong {
		t.Fatalf("NewContest(freezeMinutesExceedsDuration): expected FREEZE_TOO_LONG, got %v", err)
	}
}

func TestContest_Status(t *testing.T) {
	start := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 4, 19, 0, 0, 0, time.UTC)

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
		false, false,
		shared.RestoreGroupID("g"),
		shared.RestoreUserID("o"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		nil,
		time.Now(),
		nil,
	)

	cases := []struct {
		name   string
		now    time.Time
		status contest.Status
	}{
		{"before start", start.Add(-time.Minute), contest.StatusScheduled},
		{"at start", start, contest.StatusActive},
		{"during contest", start.Add(time.Hour), contest.StatusActive},
		{"at end", end, contest.StatusFinished},
		{"after end", end.Add(time.Second), contest.StatusFinished},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.Status(tc.now); got != tc.status {
				t.Errorf("at %v: expected %s, got %s", tc.now, tc.status, got)
			}
		})
	}
}

func TestContest_Duration(t *testing.T) {
	c := validContest(t)
	if got := c.Duration(); got != 180 {
		t.Errorf("expected 180 minutes, got %d", got)
	}
}

func mustAddProblem(t *testing.T, c interface{ AddProblem(string, string, time.Time) error }, id, problemID string) {
	t.Helper()
	if err := c.AddProblem(id, problemID, fixedNow); err != nil {
		t.Fatalf("AddProblem(%q, %q): unexpected error: %v", id, problemID, err)
	}
}

func TestContest_AddProblem(t *testing.T) {
	c := validContest(t)

	mustAddProblem(t, c, "cp-1", "problem-a")
	mustAddProblem(t, c, "cp-2", "problem-b")
	mustAddProblem(t, c, "cp-3", "problem-c")

	if len(c.Problems()) != 3 {
		t.Fatalf("expected 3 problems, got %d", len(c.Problems()))
	}
	if c.Problems()[0].Order() != 1 || c.Problems()[1].Order() != 2 || c.Problems()[2].Order() != 3 {
		t.Error("problems not in sequential order")
	}
}

func TestContest_AddProblem_Deduplication(t *testing.T) {
	c := validContest(t)

	mustAddProblem(t, c, "cp-1", "problem-a")
	mustAddProblem(t, c, "cp-2", "problem-a")

	if len(c.Problems()) != 1 {
		t.Errorf("expected 1 problem after deduplication, got %d", len(c.Problems()))
	}
}

func TestContest_AddProblem_EmptyID(t *testing.T) {
	c := validContest(t)
	if err := c.AddProblem("", "problem-a", fixedNow); err == nil {
		t.Error("expected error for empty id, got nil")
	}
	if err := c.AddProblem("cp-1", "", fixedNow); err == nil {
		t.Error("expected error for empty problemID, got nil")
	}
	if len(c.Problems()) != 0 {
		t.Error("no problem should be added on empty-id error")
	}
}

func TestContest_RemoveProblem(t *testing.T) {
	cases := []struct {
		name        string
		remove      string
		wantOrders  []int
	}{
		{"middle element", "problem-b", []int{1, 2}},
		{"first element", "problem-a", []int{1, 2}},
		{"last element", "problem-c", []int{1, 2}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := validContest(t)
			mustAddProblem(t, c, "cp-1", "problem-a")
			mustAddProblem(t, c, "cp-2", "problem-b")
			mustAddProblem(t, c, "cp-3", "problem-c")

			removed := c.RemoveProblem(tc.remove, fixedNow)
			if !removed {
				t.Fatalf("expected RemoveProblem(%q) to return true", tc.remove)
			}
			if len(c.Problems()) != len(tc.wantOrders) {
				t.Fatalf("expected %d problems, got %d", len(tc.wantOrders), len(c.Problems()))
			}
			for i, wantOrder := range tc.wantOrders {
				if got := c.Problems()[i].Order(); got != wantOrder {
					t.Errorf("problems[%d].Order() = %d, want %d", i, got, wantOrder)
				}
			}
		})
	}
}

func TestContest_RemoveProblem_NotFound(t *testing.T) {
	c := validContest(t)
	removed := c.RemoveProblem("non-existent", fixedNow)
	if removed {
		t.Error("expected RemoveProblem to return false for non-existent problem")
	}
}

func TestContest_MutationsUpdateUpdatedAt(t *testing.T) {
	mutationTime := fixedNow.Add(time.Hour)

	c := validContest(t)
	if c.UpdatedAt() != nil {
		t.Fatal("new contest should have nil updatedAt")
	}

	mustAddProblem(t, c, "cp-1", "problem-a")
	// mustAddProblem uses fixedNow; call directly with mutationTime to check tracking
	if err := c.AddProblem("cp-2", "problem-b", mutationTime); err != nil {
		t.Fatalf("AddProblem: %v", err)
	}
	if c.UpdatedAt() == nil || !c.UpdatedAt().Equal(mutationTime.UTC()) {
		t.Errorf("AddProblem: updatedAt = %v, want %v", c.UpdatedAt(), mutationTime.UTC())
	}

	removeTime := fixedNow.Add(2 * time.Hour)
	c.RemoveProblem("problem-b", removeTime)
	if c.UpdatedAt() == nil || !c.UpdatedAt().Equal(removeTime.UTC()) {
		t.Errorf("RemoveProblem: updatedAt = %v, want %v", c.UpdatedAt(), removeTime.UTC())
	}
}

func TestProblemsNeverNil(t *testing.T) {
	c := validContest(t)
	if c.Problems() == nil {
		t.Error("NewContest: Problems() must not be nil")
	}

	restored := contest.RestoreContest(
		"id",
		mustName(t, "Test"),
		nil,
		fixedNow.Add(time.Hour),
		fixedNow.Add(4*time.Hour),
		mustPenalty(t, 20),
		0,
		false,
		false,
		false, false,
		shared.RestoreGroupID("g"),
		shared.RestoreUserID("o"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		nil,
		fixedNow,
		nil,
	)
	if restored.Problems() == nil {
		t.Error("RestoreContest: Problems() must not be nil when passed nil")
	}
}

func restoredContestUpdatedAt(t *testing.T, updatedAt *time.Time) *contest.Contest {
	t.Helper()
	base := time.Date(2026, 5, 4, 14, 0, 0, 0, time.UTC)
	return contest.RestoreContest(
		"id",
		mustName(t, "Test"),
		nil,
		base.Add(time.Hour),
		base.Add(4*time.Hour),
		mustPenalty(t, 20),
		0,
		false,
		false,
		false, false,
		shared.RestoreGroupID("g"),
		shared.RestoreUserID("o"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		nil,
		base,
		updatedAt,
	)
}

func TestConcurrencyToken_IsTheUpdatedAtTheContestWasLoadedWith(t *testing.T) {
	loaded := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)

	c := restoredContestUpdatedAt(t, &loaded)

	if got := c.ConcurrencyToken(); got == nil || !got.Equal(loaded) {
		t.Errorf("ConcurrencyToken() = %v, want %v", got, loaded)
	}
}

func TestConcurrencyToken_IsNilForAContestNeverUpdated(t *testing.T) {
	c := restoredContestUpdatedAt(t, nil)

	if got := c.ConcurrencyToken(); got != nil {
		t.Errorf("ConcurrencyToken() = %v, want nil", got)
	}
}

func TestConcurrencyToken_DoesNotMoveWhenTheContestIsModified(t *testing.T) {
	loaded := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	c := restoredContestUpdatedAt(t, &loaded)

	c.SetLocked(true, loaded.Add(time.Minute))
	c.SetProblems(nil, loaded.Add(2*time.Minute))

	if got := c.UpdatedAt(); got == nil || !got.Equal(loaded.Add(2*time.Minute)) {
		t.Fatalf("UpdatedAt() = %v, want it to move to the last modification", got)
	}
	if got := c.ConcurrencyToken(); got == nil || !got.Equal(loaded) {
		t.Errorf("ConcurrencyToken() = %v, want it to stay at %v", got, loaded)
	}
}

func TestConcurrencyToken_IsNotSharedWithTheCallerThroughTheReturnedPointer(t *testing.T) {
	loaded := time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
	c := restoredContestUpdatedAt(t, &loaded)

	*c.ConcurrencyToken() = loaded.Add(time.Hour)

	if got := c.ConcurrencyToken(); !got.Equal(loaded) {
		t.Errorf("ConcurrencyToken() = %v after the caller changed the returned value, want %v", got, loaded)
	}
}

func restoredContestWithWindow(t *testing.T, start, end time.Time, allowLate bool) *contest.Contest {
	t.Helper()
	return contest.RestoreContest(
		"id",
		mustName(t, "Test"),
		nil,
		start, end,
		mustPenalty(t, 20),
		0,
		false,
		false,
		false,
		allowLate,
		shared.RestoreGroupID("g"),
		shared.RestoreUserID("o"),
		contest.RestoreParticipationMode("INDIVIDUAL"), contest.RestoreTeamSize(2, 5),
		nil,
		start.Add(-time.Hour),
		nil,
	)
}

func TestCheckRegistrationOpen(t *testing.T) {
	start := fixedNow.Add(-time.Hour)
	end := fixedNow.Add(time.Hour)

	tests := []struct {
		name      string
		now       time.Time
		allowLate bool
		wantCode  string
	}{
		{"scheduled", start.Add(-time.Minute), false, ""},
		{"scheduled with late registration", start.Add(-time.Minute), true, ""},
		{"active without late registration", fixedNow, false, contest.ErrCodeContestAlreadyStarted},
		{"active with late registration", fixedNow, true, ""},
		{"finished", end.Add(time.Minute), false, contest.ErrCodeContestNotOpenForRegistration},
		{"finished with late registration", end.Add(time.Minute), true, contest.ErrCodeContestNotOpenForRegistration},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := restoredContestWithWindow(t, start, end, tt.allowLate)

			err := c.CheckRegistrationOpen(tt.now)

			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("expected registration to be open, got %v", err)
				}
				return
			}
			var appErr *apperror.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("expected *apperror.AppError, got %v", err)
			}
			if appErr.Code != tt.wantCode || appErr.Kind != apperror.KindConflict {
				t.Errorf("got %s/%v, want 409 %s", appErr.Code, appErr.Kind, tt.wantCode)
			}
		})
	}
}

func TestSetAllowLateRegistration(t *testing.T) {
	c := validContest(t)
	if c.AllowLateRegistration() {
		t.Fatal("a new contest must not allow late registration by default")
	}

	c.SetAllowLateRegistration(true, fixedNow)

	if !c.AllowLateRegistration() {
		t.Error("AllowLateRegistration() = false after enabling it")
	}
	if c.UpdatedAt() == nil || !c.UpdatedAt().Equal(fixedNow) {
		t.Errorf("UpdatedAt() = %v, want %v", c.UpdatedAt(), fixedNow)
	}
}
