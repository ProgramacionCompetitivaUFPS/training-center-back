package material

import (
	"context"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListMaterialAuthorsUseCase(repo *mockMaterialRepository, vis *mockGroupVisibilityProvider, mem *mockGroupMemberProvider, author *mockAuthorProvider) *ListMaterialAuthorsUseCase {
	return NewListMaterialAuthorsUseCase(repo, vis, mem, author)
}

func defaultListAuthorsInput() ListMaterialAuthorsInput {
	return ListMaterialAuthorsInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
	}
}

func TestListMaterialAuthors_GroupNotFound_Returns404(t *testing.T) {
	uc := newListMaterialAuthorsUseCase(&mockMaterialRepository{}, groupVisibilityNotFound(), notLead(), stubAuthorProvider())
	_, err := uc.Execute(context.Background(), defaultListAuthorsInput())
	assertErrCode(t, err, ErrCodeGroupNotFound)
}

func TestListMaterialAuthors_NotVisibleGroup_NonMember_Returns403(t *testing.T) {
	uc := newListMaterialAuthorsUseCase(&mockMaterialRepository{}, notVisibleGroup(), notLead(), stubAuthorProvider())
	in := defaultListAuthorsInput()
	in.CurrentUser = asCoach(testOtherID)
	_, err := uc.Execute(context.Background(), in)
	assertErrCode(t, err, ErrCodeInsufficientPermissions)
}

func TestListMaterialAuthors_Lead_SeesAllStatusesAndResolvesDisplays(t *testing.T) {
	var capturedStatuses []domainMaterial.Status
	var capturedSampleSize int
	repo := &mockMaterialRepository{
		distinctAuthorIDsFn: func(_ context.Context, _ string, statuses []domainMaterial.Status, sampleSize int) ([]string, error) {
			capturedStatuses = statuses
			capturedSampleSize = sampleSize
			return []string{testAuthorID, testOtherID}, nil
		},
	}
	author := &mockAuthorProvider{
		getDisplaysFn: func(_ context.Context, userIDs []string) (map[string]*AuthorDisplay, error) {
			out := make(map[string]*AuthorDisplay, len(userIDs))
			for _, id := range userIDs {
				out[id] = &AuthorDisplay{Nickname: "nick-" + id, Name: "Name " + id}
			}
			return out, nil
		},
	}
	uc := newListMaterialAuthorsUseCase(repo, visibleGroup(), isLead(), author)

	out, err := uc.Execute(context.Background(), defaultListAuthorsInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedStatuses != nil {
		t.Errorf("expected no status constraint for a lead, got %v", capturedStatuses)
	}
	if capturedSampleSize != authorSuggestionSampleSize {
		t.Errorf("expected sampleSize %d, got %d", authorSuggestionSampleSize, capturedSampleSize)
	}
	if len(out.Authors) != 2 {
		t.Fatalf("expected 2 authors, got %d: %v", len(out.Authors), out.Authors)
	}
	if out.Authors[0].Nickname != "nick-"+testAuthorID || out.Authors[1].Nickname != "nick-"+testOtherID {
		t.Errorf("unexpected authors: %v", out.Authors)
	}
}

func TestListMaterialAuthors_Admin_SeesAllStatuses(t *testing.T) {
	var capturedStatuses []domainMaterial.Status
	repo := &mockMaterialRepository{
		distinctAuthorIDsFn: func(_ context.Context, _ string, statuses []domainMaterial.Status, _ int) ([]string, error) {
			capturedStatuses = statuses
			return []string{testAuthorID}, nil
		},
	}
	uc := newListMaterialAuthorsUseCase(repo, visibleGroup(), notLead(), stubAuthorProvider())
	in := defaultListAuthorsInput()
	in.CurrentUser = asAdmin(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedStatuses != nil {
		t.Errorf("expected no status constraint for an admin, got %v", capturedStatuses)
	}
}

func TestListMaterialAuthors_NonLeadMember_FiltersToPublished(t *testing.T) {
	var capturedStatuses []domainMaterial.Status
	repo := &mockMaterialRepository{
		distinctAuthorIDsFn: func(_ context.Context, _ string, statuses []domainMaterial.Status, _ int) ([]string, error) {
			capturedStatuses = statuses
			return []string{testAuthorID}, nil
		},
	}
	uc := newListMaterialAuthorsUseCase(repo, visibleGroup(), isMemberNotLead(), stubAuthorProvider())
	in := defaultListAuthorsInput()
	in.CurrentUser = asContestant(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(capturedStatuses) != 1 || !capturedStatuses[0].IsPublished() {
		t.Errorf("expected only PUBLISHED status constraint, got %v", capturedStatuses)
	}
}

func TestListMaterialAuthors_VisibleGroup_NonMember_Returns200(t *testing.T) {
	uc := newListMaterialAuthorsUseCase(&mockMaterialRepository{}, visibleGroup(), notLead(), stubAuthorProvider())
	in := defaultListAuthorsInput()
	in.CurrentUser = asContestant(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error for VISIBLE group non-member, got %v", err)
	}
}

func TestListMaterialAuthors_DisplayLookupFails_PropagatesError(t *testing.T) {
	repo := &mockMaterialRepository{
		distinctAuthorIDsFn: func(_ context.Context, _ string, _ []domainMaterial.Status, _ int) ([]string, error) {
			return []string{testAuthorID}, nil
		},
	}
	author := &mockAuthorProvider{
		getDisplaysFn: func(_ context.Context, _ []string) (map[string]*AuthorDisplay, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newListMaterialAuthorsUseCase(repo, visibleGroup(), isLead(), author)

	_, err := uc.Execute(context.Background(), defaultListAuthorsInput())
	if err == nil {
		t.Fatal("expected an error when author display lookup fails")
	}
}
