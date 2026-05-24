package contest

import (
	"context"
	"errors"
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newUnregisterUseCase(
	repo *mockContestRepository,
	reg *mockRegistrationRepository,
	member *mockGroupMemberProvider,
) *UnregisterFromContestUseCase {
	return NewUnregisterFromContestUseCase(repo, reg, member)
}

func validUnregisterInput() UnregisterFromContestInput {
	return UnregisterFromContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	}
}

func TestUnregisterFromContest_HappyPath(t *testing.T) {
	uc := newUnregisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead())

	err := uc.Execute(context.Background(), validUnregisterInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnregisterFromContest_ContestNotFound(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, nil
		},
	}
	uc := newUnregisterUseCase(repo, mockRegistrations(), isMemberNotLead())

	err := uc.Execute(context.Background(), validUnregisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND, got %v", err)
	}
}

func TestUnregisterFromContest_GroupMismatch_Returns404(t *testing.T) {
	uc := newUnregisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead())

	in := validUnregisterInput()
	in.GroupID = "different-group-id"

	err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND on group mismatch, got %v", err)
	}
}

func TestUnregisterFromContest_FinishedContest_Returns400(t *testing.T) {
	uc := newUnregisterUseCase(repoWith(newFinishedContest(otherID)), mockRegistrations(), isMemberNotLead())

	err := uc.Execute(context.Background(), validUnregisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeCannotUnregisterAfterStart {
		t.Errorf("expected CANNOT_UNREGISTER_AFTER_START, got %v", err)
	}
}

func TestUnregisterFromContest_NonMember_Returns403(t *testing.T) {
	uc := newUnregisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead())

	err := uc.Execute(context.Background(), validUnregisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotGroupMember {
		t.Errorf("expected NOT_GROUP_MEMBER, got %v", err)
	}
}

func TestUnregisterFromContest_NotRegistered_Returns404(t *testing.T) {
	reg := &mockRegistrationRepository{
		deleteFn: func(_ context.Context, _, _ string) error {
			return apperror.NewNotFound(domainContest.ErrCodeNotRegistered, "you are not registered to this contest")
		},
	}
	uc := newUnregisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	err := uc.Execute(context.Background(), validUnregisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeNotRegistered {
		t.Errorf("expected NOT_REGISTERED, got %v", err)
	}
}

func TestUnregisterFromContest_RepoFindError_Propagates(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newUnregisterUseCase(repo, mockRegistrations(), isMemberNotLead())

	if err := uc.Execute(context.Background(), validUnregisterInput()); err == nil {
		t.Fatal("expected error from failed FindByID")
	}
}

func TestUnregisterFromContest_MemberProviderError_Propagates(t *testing.T) {
	member := &mockGroupMemberProvider{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newUnregisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), member)

	if err := uc.Execute(context.Background(), validUnregisterInput()); err == nil {
		t.Fatal("expected error from failed IsMemberOfGroup")
	}
}

func TestUnregisterFromContest_DeleteError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		deleteFn: func(_ context.Context, _, _ string) error {
			return apperror.NewInternal()
		},
	}
	uc := newUnregisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	if err := uc.Execute(context.Background(), validUnregisterInput()); err == nil {
		t.Fatal("expected error from failed Delete")
	}
}
