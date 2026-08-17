package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGetContestUseCase(
	repo *mockContestRepository,
	group *mockGroupProvider,
	member *mockGroupMemberProvider,
	problem *mockProblemProvider,
	participant *mockContestParticipantProvider,
) *GetContestUseCase {
	return NewGetContestUseCase(repo, group, member, problem, mockOwner(), participant)
}

func validGetInput() GetContestInput {
	return GetContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	}
}

func TestGetContest_AdminSeesEverything(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFoundNotVisible(), notLead(), defaultProblemProvider(), mockParticipants())

	in := validGetInput()
	in.CurrentUser = asAdmin(callerID)

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Locked == nil {
		t.Error("admin should see locked field")
	}
	if out.Status != "SCHEDULED" {
		t.Errorf("expected SCHEDULED, got %q", out.Status)
	}
}

func TestGetContest_AdminSeesProblemsInScheduled(t *testing.T) {
	contest := newTestContestWithProblems(callerID)
	uc := newGetContestUseCase(repoWith(contest), groupFound(), notLead(), defaultProblemProvider(), mockParticipants())

	in := validGetInput()
	in.CurrentUser = asAdmin(callerID)

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Problems) == 0 {
		t.Error("admin should see problems even in SCHEDULED status")
	}
}

func TestGetContest_MemberDoesNotSeeProblemsInScheduled(t *testing.T) {
	contest := newTestContestWithProblems(callerID)
	uc := newGetContestUseCase(repoWith(contest), groupFound(), isMemberNotLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Problems) != 0 {
		t.Errorf("member should not see problems in SCHEDULED, got %d", len(out.Problems))
	}
}

func TestGetContest_LeadSeesProblemsInScheduled(t *testing.T) {
	contest := newTestContestWithProblems(callerID)
	uc := newGetContestUseCase(repoWith(contest), groupFound(), isLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Problems) == 0 {
		t.Error("lead should see problems even in SCHEDULED status")
	}
}

func TestGetContest_LeadSeesLockedField(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFound(), isLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Locked == nil {
		t.Error("lead should see locked field")
	}
}

func TestGetContest_NonLeadMemberDoesNotSeeLocked(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFound(), isMemberNotLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Locked != nil {
		t.Error("non-lead member should not see locked field")
	}
}

func TestGetContest_NotFoundIfContestNotExists(t *testing.T) {
	uc := newGetContestUseCase(&mockContestRepository{}, groupFound(), notLead(), defaultProblemProvider(), mockParticipants())

	_, err := uc.Execute(context.Background(), validGetInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND, got %v", err)
	}
}

func TestGetContest_NotFoundIfGroupMismatch(t *testing.T) {
	contest := newTestContest(callerID)
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return contest, nil
		},
	}
	uc := newGetContestUseCase(repo, groupFound(), notLead(), defaultProblemProvider(), mockParticipants())

	in := validGetInput()
	in.GroupID = "different-group-id"

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND on group mismatch, got %v", err)
	}
}

func TestGetContest_NotFoundForNonMemberInNotVisibleGroup(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFoundNotVisible(), notLead(), defaultProblemProvider(), mockParticipants())

	_, err := uc.Execute(context.Background(), validGetInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND for non-member in hidden group, got %v", err)
	}
}

func TestGetContest_MemberCanAccessNotVisibleGroup(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFoundNotVisible(), isMemberNotLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error for member in hidden group: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
}

func TestGetContest_NonMemberCanAccessVisibleGroup(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFound(), notLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error for non-member in visible group: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
}

func TestGetContest_DurationInSeconds(t *testing.T) {
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFound(), isMemberNotLead(), defaultProblemProvider(), mockParticipants())

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// testEnd - testStart = 5 hours = 18000 seconds
	expected := 5 * 60 * 60
	if out.Duration != expected {
		t.Errorf("expected duration %d seconds, got %d", expected, out.Duration)
	}
}

func TestGetContest_ParticipantCountAndRegistration(t *testing.T) {
	pp := &mockContestParticipantProvider{
		countParticipantsFn: func(_ context.Context, _ string) (int, error) { return 42, nil },
		isRegisteredFn:      func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
	uc := newGetContestUseCase(repoWith(newTestContest(callerID)), groupFound(), isMemberNotLead(), defaultProblemProvider(), pp)

	out, err := uc.Execute(context.Background(), validGetInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ParticipantCount != 42 {
		t.Errorf("expected 42 participants, got %d", out.ParticipantCount)
	}
	if !out.IsRegistered {
		t.Error("expected isRegistered=true")
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newTestContestWithProblems(ownerID string) *domainContest.Contest {
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Test Contest"),
		nil,
		testStart,
		testEnd,
		domainContest.RestorePenalty(20),
		0,
		false,
		false,
		false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(ownerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{
			domainContest.RestoreContestProblem("cp-1", testProblemID, 1),
		},
		testNow.Add(-time.Hour),
		nil,
	)
}
