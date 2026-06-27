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
	return NewRegisterToContestUseCase(repo, reg, member, mockTeamSelection())
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

	err := uc.Execute(context.Background(), validRegisterInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterToContest_ContestNotFound(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		},
	}
	uc := newRegisterUseCase(repo, mockRegistrations(), isMemberNotLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND, got %v", err)
	}
}

func TestRegisterToContest_GroupMismatch_Returns404(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead())

	in := validRegisterInput()
	in.GroupID = "different-group-id"

	err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestNotFound {
		t.Errorf("expected CONTEST_NOT_FOUND on group mismatch, got %v", err)
	}
}

func TestRegisterToContest_NonMember_Returns403(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeNotGroupMember {
		t.Errorf("expected NOT_GROUP_MEMBER, got %v", err)
	}
}

func TestRegisterToContest_Admin_Returns403(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead())

	in := validRegisterInput()
	in.CurrentUser = asAdmin(callerID)

	err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeAdminsCannotRegister {
		t.Errorf("expected ADMINS_CANNOT_REGISTER, got %v", err)
	}
}

func TestRegisterToContest_Lead_Returns403(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeLeadsCannotRegister {
		t.Errorf("expected LEADS_CANNOT_REGISTER, got %v", err)
	}
}

func TestRegisterToContest_FinishedContest_Returns400(t *testing.T) {
	uc := newRegisterUseCase(repoWith(newFinishedContest(otherID)), mockRegistrations(), isMemberNotLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainContest.ErrCodeContestAlreadyStarted {
		t.Errorf("expected CONTEST_ALREADY_STARTED, got %v", err)
	}
}

func TestRegisterToContest_AlreadyRegistered_IsIdempotent(t *testing.T) {
	reg := &mockRegistrationRepository{
		existsByContestAndUser: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	if err != nil {
		t.Errorf("expected no error for duplicate registration (idempotent), got %v", err)
	}
}

func TestRegisterToContest_RepoFindError_Propagates(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repo, mockRegistrations(), isMemberNotLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed FindByID")
	}
}

func TestRegisterToContest_MemberProviderError_Propagates(t *testing.T) {
	member := &mockGroupMemberProvider{
		getRoleFn: func(_ context.Context, _, _ string) (*string, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), member)

	err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed GetMemberRole")
	}
}

func TestRegisterToContest_RegistrationCheckError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		existsByContestAndUser: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed registration check")
	}
}

func TestRegisterToContest_AlreadyInTeam_Returns409(t *testing.T) {
	checker := &mockTeamSelectionChecker{
		fn: func(_, _ string) (bool, error) { return true, nil },
	}
	uc := NewRegisterToContestUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead(), checker)

	err := uc.Execute(context.Background(), validRegisterInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeAlreadyInTeam {
		t.Errorf("expected ALREADY_IN_TEAM, got %v", err)
	}
}

func TestRegisterToContest_TeamSelectionCheckerError_Propagates(t *testing.T) {
	checker := &mockTeamSelectionChecker{
		fn: func(_, _ string) (bool, error) { return false, apperror.NewInternal() },
	}
	uc := NewRegisterToContestUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead(), checker)

	err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed team selection check")
	}
}

func TestRegisterToContest_SaveError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		saveFn: func(_ context.Context, _ *domainContest.ContestRegistration) error {
			return apperror.NewInternal()
		},
	}
	uc := newRegisterUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead())

	err := uc.Execute(context.Background(), validRegisterInput())

	if err == nil {
		t.Fatal("expected error from failed Save")
	}
}
