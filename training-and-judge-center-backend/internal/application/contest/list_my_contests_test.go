package contest

import (
	"context"
	"testing"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

func newListMyContestsUseCase(
	repo *mockContestRepository,
	group *mockGroupProvider,
	participant *mockContestParticipantProvider,
) *ListMyContestsUseCase {
	return NewListMyContestsUseCase(repo, group, participant)
}

func validMyContestsInput() ListMyContestsInput {
	return ListMyContestsInput{
		CurrentUser: asCoach(callerID),
		Page:        1,
		Limit:       20,
	}
}

func TestListMyContests_Admin_NoGroupRestriction(t *testing.T) {
	// groupProvider.ListAccessibleGroupIDs defaults to returning nil for
	// admins (see mockGroupProvider); assert that nil sentinel reaches the
	// repo untouched, meaning "no group filter" rather than "zero groups".
	var capturedGroupIDs []string
	repo := &mockContestRepository{
		listByGroupIDsFn: func(_ context.Context, groupIDs []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			capturedGroupIDs = groupIDs
			return []*domainContest.Contest{newTestContest(callerID)}, 1, nil
		},
	}
	uc := newListMyContestsUseCase(repo, groupFound(), mockParticipants())

	in := validMyContestsInput()
	in.CurrentUser = asAdmin(callerID)

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedGroupIDs != nil {
		t.Errorf("expected nil groupIDs sentinel (no restriction) for admin, got %v", capturedGroupIDs)
	}
	if len(out.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(out.Items))
	}
}

func TestListMyContests_Member_ScopedToAccessibleGroups(t *testing.T) {
	var capturedGroupIDs []string
	repo := &mockContestRepository{
		listByGroupIDsFn: func(_ context.Context, groupIDs []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			capturedGroupIDs = groupIDs
			return []*domainContest.Contest{newTestContest(callerID)}, 1, nil
		},
	}
	group := &mockGroupProvider{
		listAccessibleGroupIDsFn: func(_ context.Context, _ string, _ bool) ([]string, error) {
			return []string{testGroupID}, nil
		},
	}
	uc := newListMyContestsUseCase(repo, group, mockParticipants())

	out, err := uc.Execute(context.Background(), validMyContestsInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedGroupIDs) != 1 || capturedGroupIDs[0] != testGroupID {
		t.Errorf("expected groupIDs=[%s], got %v", testGroupID, capturedGroupIDs)
	}
	if len(out.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(out.Items))
	}
}

func TestListMyContests_NoAccessibleGroups_ReturnsEmpty(t *testing.T) {
	repo := &mockContestRepository{
		listByGroupIDsFn: func(_ context.Context, groupIDs []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			if groupIDs == nil || len(groupIDs) != 0 {
				t.Errorf("expected empty (non-nil) groupIDs slice, got %v", groupIDs)
			}
			return []*domainContest.Contest{}, 0, nil
		},
	}
	group := &mockGroupProvider{
		listAccessibleGroupIDsFn: func(_ context.Context, _ string, _ bool) ([]string, error) {
			return []string{}, nil
		},
	}
	uc := newListMyContestsUseCase(repo, group, mockParticipants())

	out, err := uc.Execute(context.Background(), validMyContestsInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(out.Items))
	}
	if out.Pagination.Total != 0 {
		t.Errorf("expected total 0, got %d", out.Pagination.Total)
	}
}

func TestListMyContests_GroupNameResolved(t *testing.T) {
	repo := &mockContestRepository{
		listByGroupIDsFn: func(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			return []*domainContest.Contest{newTestContest(callerID)}, 1, nil
		},
	}
	group := &mockGroupProvider{
		findByIDsFn: func(_ context.Context, groupIDs []string) (map[string]*GroupInfo, error) {
			out := make(map[string]*GroupInfo, len(groupIDs))
			for _, id := range groupIDs {
				out[id] = &GroupInfo{ID: id, Name: "My Group", IsVisible: true}
			}
			return out, nil
		},
	}
	uc := newListMyContestsUseCase(repo, group, mockParticipants())

	out, err := uc.Execute(context.Background(), validMyContestsInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out.Items))
	}
	if out.Items[0].Group.ID != testGroupID || out.Items[0].Group.Name != "My Group" {
		t.Errorf("expected group {%s, My Group}, got %+v", testGroupID, out.Items[0].Group)
	}
}

func TestListMyContests_InvalidPageReturnsError(t *testing.T) {
	uc := newListMyContestsUseCase(&mockContestRepository{}, groupFound(), mockParticipants())

	in := validMyContestsInput()
	in.Page = 0

	_, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for page=0, got nil")
	}
}

func TestListMyContests_InvalidLimitReturnsError(t *testing.T) {
	uc := newListMyContestsUseCase(&mockContestRepository{}, groupFound(), mockParticipants())

	in := validMyContestsInput()
	in.Limit = 999

	_, err := uc.Execute(context.Background(), in)
	if err == nil {
		t.Fatal("expected error for limit=999 (exceeds MaxLimit), got nil")
	}
}

func TestListMyContests_PaginationOutput(t *testing.T) {
	repo := &mockContestRepository{
		listByGroupIDsFn: func(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			return []*domainContest.Contest{}, 55, nil
		},
	}
	uc := newListMyContestsUseCase(repo, groupFound(), mockParticipants())

	in := validMyContestsInput()
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

func TestListMyContests_StatusFilterParsed(t *testing.T) {
	var capturedFilters domainContest.ListFilters
	repo := &mockContestRepository{
		listByGroupIDsFn: func(_ context.Context, _ []string, f domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
			capturedFilters = f
			return []*domainContest.Contest{}, 0, nil
		},
	}
	uc := newListMyContestsUseCase(repo, groupFound(), mockParticipants())

	in := validMyContestsInput()
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
