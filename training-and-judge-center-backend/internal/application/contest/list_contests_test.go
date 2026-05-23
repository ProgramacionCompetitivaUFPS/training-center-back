package contest

import (
	"context"
	"errors"
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListContestsUseCase(
	repo *mockContestRepository,
	group *mockGroupProvider,
	member *mockGroupMemberProvider,
	participant *mockContestParticipantProvider,
) *ListContestsUseCase {
	return NewListContestsUseCase(repo, group, member, participant)
}

func validListInput() ListContestsInput {
	return ListContestsInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		Page:        1,
		Limit:       20,
	}
}

func TestListContests_HappyPath(t *testing.T) {
	repo := &mockContestRepository{
		listFn: func(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			return []*domainContest.Contest{newTestContest(callerID)}, 1, nil
		},
	}
	uc := newListContestsUseCase(repo, groupFound(), isMemberNotLead(), mockParticipants())

	out, err := uc.Execute(context.Background(), validListInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(out.Items))
	}
	if out.Pagination.Total != 1 {
		t.Errorf("expected total 1, got %d", out.Pagination.Total)
	}
	if out.Pagination.TotalPages != 1 {
		t.Errorf("expected 1 page, got %d", out.Pagination.TotalPages)
	}
}

func TestListContests_EmptyResult(t *testing.T) {
	uc := newListContestsUseCase(&mockContestRepository{}, groupFound(), isMemberNotLead(), mockParticipants())

	out, err := uc.Execute(context.Background(), validListInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(out.Items))
	}
}

func TestListContests_GroupNotFound(t *testing.T) {
	uc := newListContestsUseCase(&mockContestRepository{}, groupNotFound(), notLead(), mockParticipants())

	_, err := uc.Execute(context.Background(), validListInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestListContests_NotFoundForNonMemberInNotVisibleGroup(t *testing.T) {
	uc := newListContestsUseCase(&mockContestRepository{}, groupFoundNotVisible(), notLead(), mockParticipants())

	_, err := uc.Execute(context.Background(), validListInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND for non-member in hidden group, got %v", err)
	}
}

func TestListContests_MemberCanListNotVisibleGroup(t *testing.T) {
	uc := newListContestsUseCase(&mockContestRepository{}, groupFoundNotVisible(), isMemberNotLead(), mockParticipants())

	out, err := uc.Execute(context.Background(), validListInput())

	if err != nil {
		t.Fatalf("unexpected error for member in hidden group: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
}

func TestListContests_AdminCanListNotVisibleGroup(t *testing.T) {
	uc := newListContestsUseCase(&mockContestRepository{}, groupFoundNotVisible(), notLead(), mockParticipants())

	in := validListInput()
	in.CurrentUser = asAdmin(callerID)

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error for admin in hidden group: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
}

func TestListContests_InvalidPageReturnsError(t *testing.T) {
	uc := newListContestsUseCase(
		&mockContestRepository{},
		groupFound(),
		isMemberNotLead(),
		mockParticipants(),
	)

	in := validListInput()
	in.Page = 0

	_, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for page=0, got nil")
	}
}

func TestListContests_InvalidLimitReturnsError(t *testing.T) {
	uc := newListContestsUseCase(
		&mockContestRepository{},
		groupFound(),
		isMemberNotLead(),
		mockParticipants(),
	)

	in := validListInput()
	in.Limit = 999

	_, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for limit=999 (exceeds MaxLimit), got nil")
	}
}

func TestListContests_StatusFilterParsed(t *testing.T) {
	var capturedFilters domainContest.ListFilters
	repo := &mockContestRepository{
		listFn: func(_ context.Context, f domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			capturedFilters = f
			return []*domainContest.Contest{}, 0, nil
		},
	}
	uc := newListContestsUseCase(repo, groupFound(), isMemberNotLead(), mockParticipants())

	in := validListInput()
	s := "ACTIVE"
	in.Status = &s

	_, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.Status == nil || *capturedFilters.Status != domainContest.StatusActive {
		t.Errorf("expected ACTIVE status filter, got %v", capturedFilters.Status)
	}
}

func TestListContests_InvalidStatusIgnored(t *testing.T) {
	var capturedFilters domainContest.ListFilters
	repo := &mockContestRepository{
		listFn: func(_ context.Context, f domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			capturedFilters = f
			return []*domainContest.Contest{}, 0, nil
		},
	}
	uc := newListContestsUseCase(repo, groupFound(), isMemberNotLead(), mockParticipants())

	in := validListInput()
	s := "INVALID_STATUS"
	in.Status = &s

	_, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.Status != nil {
		t.Errorf("expected nil status filter for invalid value, got %v", capturedFilters.Status)
	}
}

func TestListContests_PaginationOutput(t *testing.T) {
	repo := &mockContestRepository{
		listFn: func(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			return []*domainContest.Contest{}, 55, nil
		},
	}
	uc := newListContestsUseCase(repo, groupFound(), isMemberNotLead(), mockParticipants())

	in := validListInput()
	in.Page = 2
	in.Limit = 20

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Pagination.TotalPages != 3 {
		t.Errorf("expected 3 total pages for 55 items with limit 20, got %d", out.Pagination.TotalPages)
	}
	if out.Pagination.Page != 2 {
		t.Errorf("expected page 2, got %d", out.Pagination.Page)
	}
}

func TestListContests_DurationInSeconds(t *testing.T) {
	repo := &mockContestRepository{
		listFn: func(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			return []*domainContest.Contest{newTestContest(callerID)}, 1, nil
		},
	}
	uc := newListContestsUseCase(repo, groupFound(), isMemberNotLead(), mockParticipants())

	out, err := uc.Execute(context.Background(), validListInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// testEnd - testStart = 5 hours = 18000 seconds
	expected := 5 * 60 * 60
	if out.Items[0].Duration != expected {
		t.Errorf("expected duration %d seconds, got %d", expected, out.Items[0].Duration)
	}
}
