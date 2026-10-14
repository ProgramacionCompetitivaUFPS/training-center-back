package material

import (
	"context"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
)

func newListMaterialTagsUseCase(repo *mockMaterialRepository, vis *mockGroupVisibilityProvider, mem *mockGroupMemberProvider) *ListMaterialTagsUseCase {
	return NewListMaterialTagsUseCase(repo, vis, mem)
}

func defaultListTagsInput() ListMaterialTagsInput {
	return ListMaterialTagsInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
	}
}

func TestListMaterialTags_GroupNotFound_Returns404(t *testing.T) {
	uc := newListMaterialTagsUseCase(&mockMaterialRepository{}, groupVisibilityNotFound(), notLead())
	_, err := uc.Execute(context.Background(), defaultListTagsInput())
	assertErrCode(t, err, ErrCodeGroupNotFound)
}

func TestListMaterialTags_NotVisibleGroup_NonMember_Returns403(t *testing.T) {
	uc := newListMaterialTagsUseCase(&mockMaterialRepository{}, notVisibleGroup(), notLead())
	in := defaultListTagsInput()
	in.CurrentUser = asCoach(testOtherID)
	_, err := uc.Execute(context.Background(), in)
	assertErrCode(t, err, ErrCodeInsufficientPermissions)
}

func TestListMaterialTags_Lead_SeesAllStatuses(t *testing.T) {
	var capturedStatuses []domainMaterial.Status
	var capturedSampleSize int
	repo := &mockMaterialRepository{
		distinctTagsFn: func(_ context.Context, _ string, statuses []domainMaterial.Status, sampleSize int) ([]string, error) {
			capturedStatuses = statuses
			capturedSampleSize = sampleSize
			return []string{"tutorial", "midterm-2026"}, nil
		},
	}
	uc := newListMaterialTagsUseCase(repo, visibleGroup(), isLead())

	out, err := uc.Execute(context.Background(), defaultListTagsInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedStatuses != nil {
		t.Errorf("expected no status constraint for a lead, got %v", capturedStatuses)
	}
	if capturedSampleSize != tagSuggestionSampleSize {
		t.Errorf("expected sampleSize %d, got %d", tagSuggestionSampleSize, capturedSampleSize)
	}
	if len(out.Tags) != 2 || out.Tags[0] != "tutorial" || out.Tags[1] != "midterm-2026" {
		t.Errorf("unexpected tags: %v", out.Tags)
	}
}

func TestListMaterialTags_Admin_SeesAllStatuses(t *testing.T) {
	var capturedStatuses []domainMaterial.Status
	repo := &mockMaterialRepository{
		distinctTagsFn: func(_ context.Context, _ string, statuses []domainMaterial.Status, _ int) ([]string, error) {
			capturedStatuses = statuses
			return []string{"tutorial"}, nil
		},
	}
	uc := newListMaterialTagsUseCase(repo, visibleGroup(), notLead())
	in := defaultListTagsInput()
	in.CurrentUser = asAdmin(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedStatuses != nil {
		t.Errorf("expected no status constraint for an admin, got %v", capturedStatuses)
	}
}

func TestListMaterialTags_NonLeadMember_FiltersToPublished(t *testing.T) {
	var capturedStatuses []domainMaterial.Status
	repo := &mockMaterialRepository{
		distinctTagsFn: func(_ context.Context, _ string, statuses []domainMaterial.Status, _ int) ([]string, error) {
			capturedStatuses = statuses
			return []string{"tutorial"}, nil
		},
	}
	uc := newListMaterialTagsUseCase(repo, visibleGroup(), isMemberNotLead())
	in := defaultListTagsInput()
	in.CurrentUser = asContestant(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(capturedStatuses) != 1 || !capturedStatuses[0].IsPublished() {
		t.Errorf("expected only PUBLISHED status constraint, got %v", capturedStatuses)
	}
}

func TestListMaterialTags_VisibleGroup_NonMember_Returns200(t *testing.T) {
	uc := newListMaterialTagsUseCase(&mockMaterialRepository{}, visibleGroup(), notLead())
	in := defaultListTagsInput()
	in.CurrentUser = asContestant(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error for VISIBLE group non-member, got %v", err)
	}
}
