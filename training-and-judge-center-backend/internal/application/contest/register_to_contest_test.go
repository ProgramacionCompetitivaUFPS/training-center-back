package contest

import (
	"context"
	"errors"
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRegisterUseCase(
	repo *mockContestRepository,
	reg *mockRegistrationRepository,
	member *mockGroupMemberProvider,
) *RegisterToContestUseCase {
	return NewRegisterToContestUseCase(repo, reg, member)
}

func validRegisterInput() RegisterToContestInput {
	return RegisterToContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	}
}

func TestRegisterToContest_HappyPath(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead())

	out, err := uc.Execute(context.Background(), validRegisterInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || out.RegisteredAt.IsZero() {
		t.Error("expected non-zero RegisteredAt in output")
	}
}

func TestRegisterToContest_ContestNotFound(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, nil
		},
	}
	uc := newRegisterUseCase(repo, mockRegistrations(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND, got %v", err)
	}
}

func TestRegisterToContest_GroupMismatch_Returns404(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead())

	in := validRegisterInput()
	in.GroupID = "different-group-id"

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND on group mismatch, got %v", err)
	}
}

func TestRegisterToContest_NonMember_Returns403(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotGroupMember {
		t.Errorf("expected NOT_GROUP_MEMBER, got %v", err)
	}
}

func TestRegisterToContest_AdminBypassesMembership(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead())

	in := validRegisterInput()
	in.CurrentUser = asAdmin(callerID)

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("admin should bypass membership check, got %v", err)
	}
	if out == nil {
		t.Error("expected output")
	}
}

func TestRegisterToContest_FinishedContest_Returns409(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newFinishedContest(otherID)), mockRegistrations(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeRegistrationClosed {
		t.Errorf("expected REGISTRATION_CLOSED, got %v", err)
	}
}

func TestRegisterToContest_AlreadyRegistered_Returns409(t *testing.T) {
	reg := &mockRegistrationRepository{
		existsByContestAndUser: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeAlreadyRegistered {
		t.Errorf("expected ALREADY_REGISTERED, got %v", err)
	}
}

func TestRegisterToContest_RepoFindError_Propagates(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repo, mockRegistrations(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed FindByID")
	}
}

func TestRegisterToContest_MemberProviderError_Propagates(t *testing.T) {
	member := &mockGroupMemberProvider{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), member)

	_, err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed IsMemberOfGroup")
	}
}

func TestRegisterToContest_RegistrationCheckError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		existsByContestAndUser: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed registration check")
	}
}

func TestRegisterToContest_SaveError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		saveFn: func(_ context.Context, _ *domainContest.ContestRegistration) error {
			return apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	_, err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed Save")
	}
}
