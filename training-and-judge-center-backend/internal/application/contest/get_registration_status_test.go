package contest

import (
	"context"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)


func newGetStatusUseCase(
	repo *mockContestRepository,
	reg *mockRegistrationRepository,
) *GetRegistrationStatusUseCase {
	return NewGetRegistrationStatusUseCase(repo, reg)
}

func validStatusInput() GetRegistrationStatusInput {
	return GetRegistrationStatusInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
	}
}

func TestGetRegistrationStatus_Registered(t *testing.T) {
	reg := &mockRegistrationRepository{
		findByContestAndUserFn: func(_ context.Context, _, _ string) (*domainContest.ContestRegistration, error) {
			return domainContest.RestoreContestRegistration("r1", testContestID, callerID, time.Now()), nil
		},
	}
	uc := newGetStatusUseCase(repoWith(newTestContest(otherID)), reg)

	out, err := uc.Execute(context.Background(), validStatusInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Registered {
		t.Error("expected registered=true")
	}
	if out.RegisteredAt == nil {
		t.Error("expected non-nil RegisteredAt")
	}
}

func TestGetRegistrationStatus_NotRegistered(t *testing.T) {
	uc := newGetStatusUseCase(repoWith(newTestContest(otherID)), mockRegistrations())

	out, err := uc.Execute(context.Background(), validStatusInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Registered {
		t.Error("expected registered=false")
	}
	if out.RegisteredAt != nil {
		t.Error("expected nil RegisteredAt")
	}
}

func TestGetRegistrationStatus_ContestNotFound(t *testing.T) {
	uc := newGetStatusUseCase(&mockContestRepository{}, mockRegistrations())

	_, err := uc.Execute(context.Background(), validStatusInput())

	if err == nil {
		t.Fatal("expected error for missing contest")
	}
}

func TestGetRegistrationStatus_GroupMismatch_Returns404(t *testing.T) {
	uc := newGetStatusUseCase(repoWith(newTestContest(otherID)), mockRegistrations())

	in := validStatusInput()
	in.GroupID = "wrong-group"

	_, err := uc.Execute(context.Background(), in)

	if err == nil {
		t.Fatal("expected NOT_FOUND on group mismatch")
	}
}

func TestGetRegistrationStatus_RepoFindError_Propagates(t *testing.T) {
	repo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newGetStatusUseCase(repo, mockRegistrations())

	if _, err := uc.Execute(context.Background(), validStatusInput()); err == nil {
		t.Fatal("expected error from failed FindByID")
	}
}

func TestGetRegistrationStatus_RegistrationLookupError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		findByContestAndUserFn: func(_ context.Context, _, _ string) (*domainContest.ContestRegistration, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newGetStatusUseCase(repoWith(newTestContest(otherID)), reg)

	if _, err := uc.Execute(context.Background(), validStatusInput()); err == nil {
		t.Fatal("expected error from failed FindByContestAndUser")
	}
}
