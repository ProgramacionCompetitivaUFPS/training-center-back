package contest

import (
	"context"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListRegistrationsUseCase(
	repo *mockContestRepository,
	reg *mockRegistrationRepository,
	member *mockGroupMemberProvider,
	nicknames *mockNicknameProvider,
) *ListContestRegistrationsUseCase {
	return NewListContestRegistrationsUseCase(repo, reg, member, nicknames)
}

func validListRegistrationsInput() ListContestRegistrationsInput {
	return ListContestRegistrationsInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       50,
	}
}

func TestListContestRegistrations_HappyPath(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return []*domainContest.ContestRegistration{
				domainContest.RestoreContestRegistration("r1", testContestID, callerID, time.Now()),
			}, 1, nil
		},
	}
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead(), mockNicknames())

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 1 {
		t.Errorf("expected total=1, got %d", out.Total)
	}
	if len(out.Registrations) != 1 {
		t.Fatalf("expected 1 registration, got %d", len(out.Registrations))
	}
	if out.Registrations[0].Nickname == "" {
		t.Error("expected non-empty nickname")
	}
}

func TestListContestRegistrations_EmptyList(t *testing.T) {
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isMemberNotLead(), mockNicknames())

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 0 || len(out.Registrations) != 0 {
		t.Errorf("expected empty list, got total=%d len=%d", out.Total, len(out.Registrations))
	}
}

func TestListContestRegistrations_ContestNotFound(t *testing.T) {
	uc := newListRegistrationsUseCase(&mockContestRepository{}, mockRegistrations(), isMemberNotLead(), mockNicknames())

	if _, err := uc.Execute(context.Background(), validListRegistrationsInput()); err == nil {
		t.Fatal("expected error for missing contest")
	}
}

func TestListContestRegistrations_NonMember_Returns403(t *testing.T) {
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), notLead(), mockNicknames())

	_, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err == nil {
		t.Fatal("expected 403 for non-member")
	}
}

func TestListContestRegistrations_LeadCanView(t *testing.T) {
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), mockRegistrations(), isLead(), mockNicknames())

	out, err := uc.Execute(context.Background(), validListRegistrationsInput())

	if err != nil {
		t.Fatalf("lead should be able to view registrations, got %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
}

func TestListContestRegistrations_NicknameProviderError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return []*domainContest.ContestRegistration{
				domainContest.RestoreContestRegistration("r1", testContestID, callerID, time.Now()),
			}, 1, nil
		},
	}
	nicknames := &mockNicknameProvider{
		getFn: func(_ context.Context, _ []string) (map[string]string, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead(), nicknames)

	if _, err := uc.Execute(context.Background(), validListRegistrationsInput()); err == nil {
		t.Fatal("expected error from failed GetNicknamesByIDs")
	}
}

func TestListContestRegistrations_ListError_Propagates(t *testing.T) {
	reg := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return nil, 0, apperror.NewInternal()
		},
	}
	uc := newListRegistrationsUseCase(repoWith(newTestContest(otherID)), reg, isMemberNotLead(), mockNicknames())

	if _, err := uc.Execute(context.Background(), validListRegistrationsInput()); err == nil {
		t.Fatal("expected error from failed ListByContest")
	}
}
