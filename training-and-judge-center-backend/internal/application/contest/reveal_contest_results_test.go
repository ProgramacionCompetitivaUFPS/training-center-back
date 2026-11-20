package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// finishedFrozenContest ended an hour ago and had a 60-minute freeze.
func finishedFrozenContest(revealedAt *time.Time) *domainContest.Contest {
	now := time.Now()
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Frozen Contest"),
		nil,
		now.Add(-6*time.Hour),
		now.Add(-time.Hour),
		domainContest.RestorePenalty(20),
		60, false, false, false, false,
		revealedAt,
		nil,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func newRevealUseCase(c *domainContest.Contest, members *mockGroupMemberProvider, saved **domainContest.Contest) *RevealContestResultsUseCase {
	return NewRevealContestResultsUseCase(
		&mockContestRepository{
			findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) { return c, nil },
			updateFn: func(_ context.Context, updated *domainContest.Contest) error {
				if saved != nil {
					*saved = updated
				}
				return nil
			},
		},
		members,
		&mockTransactionManager{},
	)
}

func revealInputFor(user appshared.CurrentUser) RevealContestResultsInput {
	return RevealContestResultsInput{CurrentUser: user, GroupID: testGroupID, ContestID: testContestID, Now: time.Now()}
}

func assertRevealRejected(t *testing.T, err error, wantCode string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != wantCode {
		t.Fatalf("expected %s, got %v", wantCode, err)
	}
}

func TestRevealContestResults_LeadAndAdmin_RevealAndPersist(t *testing.T) {
	cases := map[string]struct {
		user    appshared.CurrentUser
		members *mockGroupMemberProvider
	}{
		"lead":  {asCoach(callerID), isLead()},
		"admin": {asAdmin(callerID), notLead()},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var saved *domainContest.Contest
			uc := newRevealUseCase(finishedFrozenContest(nil), tc.members, &saved)

			if err := uc.Execute(context.Background(), revealInputFor(tc.user)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if saved == nil || saved.ResultsRevealedAt() == nil {
				t.Fatal("the revealed contest must be persisted")
			}
		})
	}
}

func TestRevealContestResults_NotLead_Returns403(t *testing.T) {
	uc := newRevealUseCase(finishedFrozenContest(nil), isMemberNotLead(), nil)

	err := uc.Execute(context.Background(), revealInputFor(asContestant(callerID)))

	assertRevealRejected(t, err, ErrCodeInsufficientPermissions)
}

func TestRevealContestResults_AlreadyRevealed_DoesNotPersistAgain(t *testing.T) {
	first := time.Now().Add(-time.Minute)
	var saved *domainContest.Contest
	uc := newRevealUseCase(finishedFrozenContest(&first), isLead(), &saved)

	if err := uc.Execute(context.Background(), revealInputFor(asCoach(callerID))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved != nil {
		t.Error("an already revealed contest must not be saved again")
	}
}

func TestRevealContestResults_ContestStillRunning_Returns409(t *testing.T) {
	running := domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Running"),
		nil,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
		domainContest.RestorePenalty(20),
		30, false, false, false, false,
		nil,
		nil,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
	uc := newRevealUseCase(running, isLead(), nil)

	err := uc.Execute(context.Background(), revealInputFor(asCoach(callerID)))

	assertRevealRejected(t, err, domainContest.ErrCodeContestNotFinished)
}

func TestRevealContestResults_WrongGroup_Returns404(t *testing.T) {
	uc := newRevealUseCase(finishedFrozenContest(nil), isLead(), nil)
	in := revealInputFor(asCoach(callerID))
	in.GroupID = "another-group"

	err := uc.Execute(context.Background(), in)

	assertRevealRejected(t, err, domainContest.ErrCodeContestNotFound)
}

func TestGetStandingsUseCase_FinishedContestStaysFrozenUntilRevealed(t *testing.T) {
	cache := func() *mockStandingsCache {
		return boardCache(nil,
			boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(90 * time.Minute)}),
		)
	}

	frozen := executeBoard(t, boardContest(finishedFrozenContest(nil)), cache(), nil)
	if !frozen.Meta.IsFrozen || frozen.Contest.ResultsRevealedAt != nil {
		t.Errorf("finished contest not revealed: isFrozen=%v revealedAt=%v, want frozen and no reveal", frozen.Meta.IsFrozen, frozen.Contest.ResultsRevealedAt)
	}
	if a := headerFor(t, frozen, boardProblemA); a.Accepted != 0 {
		t.Errorf("problem A accepted = %d, want 0 while the results are not revealed", a.Accepted)
	}

	revealedAt := time.Now().Add(-time.Minute)
	revealed := executeBoard(t, boardContest(finishedFrozenContest(&revealedAt)), cache(), nil)
	if revealed.Meta.IsFrozen || revealed.Contest.ResultsRevealedAt == nil {
		t.Errorf("revealed contest: isFrozen=%v revealedAt=%v, want unfrozen with the reveal instant", revealed.Meta.IsFrozen, revealed.Contest.ResultsRevealedAt)
	}
	if a := headerFor(t, revealed, boardProblemA); a.Accepted != 1 {
		t.Errorf("problem A accepted = %d, want 1 once revealed", a.Accepted)
	}
}

func TestGetStandingsUseCase_LeadSeesRealtimeOnFinishedContestBeforeReveal(t *testing.T) {
	cache := boardCache(nil,
		boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(90 * time.Minute)}),
	)
	uc := newGetStandingsUseCase(boardContest(finishedFrozenContest(nil)), nil, nil, cache, visibleGroup(), isLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asCoach(callerID), GroupID: testGroupID, ContestID: testContestID,
		Realtime: true, Page: 1, Limit: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a := headerFor(t, out, boardProblemA); a.Accepted != 1 {
		t.Errorf("problem A accepted = %d, want 1 in real time", a.Accepted)
	}
}

func TestRevealContestResults_PastAutoRevealTime_DoesNotPersist(t *testing.T) {
	c := finishedFrozenContest(nil)
	minutes := 0 // opens as soon as the contest ended, an hour ago
	if err := c.SetAutoRevealMinutes(&minutes, testNow); err != nil {
		t.Fatal(err)
	}
	var saved *domainContest.Contest
	uc := newRevealUseCase(c, isLead(), &saved)

	if err := uc.Execute(context.Background(), revealInputFor(asCoach(callerID))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved != nil {
		t.Error("results already open by themselves: nothing must be saved")
	}
}

func TestGetStandingsUseCase_FinishedContestOpensByItselfAtTheAutoRevealTime(t *testing.T) {
	// the contest ended an hour ago
	run := func(autoRevealMinutes int) *GetStandingsOutput {
		c := finishedFrozenContest(nil)
		if err := c.SetAutoRevealMinutes(&autoRevealMinutes, testNow); err != nil {
			t.Fatal(err)
		}
		cache := boardCache(nil,
			boardParticipant("u1", map[string]domainContest.ProblemAttempt{boardProblemA: solvedAt(90 * time.Minute)}),
		)
		return executeBoard(t, boardContest(c), cache, nil)
	}

	waiting := run(120)
	if !waiting.Meta.IsFrozen || waiting.Contest.AutoRevealAt == nil {
		t.Errorf("auto reveal in an hour: isFrozen=%v autoRevealAt=%v, want frozen with the auto reveal time", waiting.Meta.IsFrozen, waiting.Contest.AutoRevealAt)
	}
	opened := run(30)
	if opened.Meta.IsFrozen {
		t.Error("the auto reveal time has passed: the standings must be unfrozen")
	}
	if a := headerFor(t, opened, boardProblemA); a.Accepted != 1 {
		t.Errorf("problem A accepted = %d, want 1 once open", a.Accepted)
	}
}
