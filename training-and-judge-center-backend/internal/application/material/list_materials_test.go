package material

import (
	"context"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListUC(repo *mockMaterialRepository, vis *mockGroupVisibilityProvider, mem *mockGroupMemberProvider) *ListMaterialsUseCase {
	return NewListMaterialsUseCase(repo, vis, mem, stubAuthorProvider())
}

func defaultListInput() ListMaterialsInput {
	return ListMaterialsInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		Page:        1,
		Limit:       20,
	}
}

func TestListMaterials_InvalidPage_Returns400(t *testing.T) {
	uc := newListUC(&mockMaterialRepository{}, visibleGroup(), isMemberNotLead())
	in := defaultListInput()
	in.Page = 0
	_, err := uc.Execute(context.Background(), in)
	assertErrCode(t, err, "VALIDATION_ERROR")
}

func TestListMaterials_InvalidLimit_Returns400(t *testing.T) {
	uc := newListUC(&mockMaterialRepository{}, visibleGroup(), isMemberNotLead())

	for _, limit := range []int{0, 101} {
		in := defaultListInput()
		in.Limit = limit
		_, err := uc.Execute(context.Background(), in)
		assertErrCode(t, err, "VALIDATION_ERROR")
	}
}

func TestListMaterials_GroupNotFound_Returns404(t *testing.T) {
	uc := newListUC(&mockMaterialRepository{}, groupVisibilityNotFound(), notLead())
	_, err := uc.Execute(context.Background(), defaultListInput())
	assertErrCode(t, err, ErrCodeGroupNotFound)
}

func TestListMaterials_NotVisibleGroup_NonMember_Returns403(t *testing.T) {
	uc := newListUC(&mockMaterialRepository{}, notVisibleGroup(), notLead())
	in := defaultListInput()
	in.CurrentUser = asCoach(testOtherID)
	_, err := uc.Execute(context.Background(), in)
	assertErrCode(t, err, ErrCodeInsufficientPerms)
}

func TestListMaterials_VisibleGroup_NonMember_Returns200(t *testing.T) {
	uc := newListUC(repoWithList(nil), visibleGroup(), notLead())
	in := defaultListInput()
	in.CurrentUser = asContestant(testOtherID)

	_, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("expected no error for VISIBLE group non-member, got %v", err)
	}
}

func TestListMaterials_Member_FiltersOnlyPublished(t *testing.T) {
	var capturedFilters domainMaterial.ListFilters
	repo := &mockMaterialRepository{
		listFn: func(_ context.Context, _ string, f domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
			capturedFilters = f
			return nil, 0, nil
		},
	}
	uc := newListUC(repo, visibleGroup(), isMemberNotLead())
	in := defaultListInput()
	in.CurrentUser = asContestant(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedFilters.Statuses) != 1 || capturedFilters.Statuses[0].String() != "PUBLISHED" {
		t.Errorf("expected only PUBLISHED status filter, got %v", capturedFilters.Statuses)
	}
}

func TestListMaterials_Lead_FiltersDraftAndPublished(t *testing.T) {
	var capturedFilters domainMaterial.ListFilters
	repo := &mockMaterialRepository{
		listFn: func(_ context.Context, _ string, f domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
			capturedFilters = f
			return nil, 0, nil
		},
	}
	uc := newListUC(repo, visibleGroup(), isLead())

	if _, err := uc.Execute(context.Background(), defaultListInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedFilters.Statuses) != 0 {
		t.Errorf("expected no status filter for Lead, got %v", capturedFilters.Statuses)
	}
}

func TestListMaterials_Admin_FiltersDraftAndPublished(t *testing.T) {
	var capturedFilters domainMaterial.ListFilters
	repo := &mockMaterialRepository{
		listFn: func(_ context.Context, _ string, f domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
			capturedFilters = f
			return nil, 0, nil
		},
	}
	uc := newListUC(repo, visibleGroup(), notLead())
	in := defaultListInput()
	in.CurrentUser = asAdmin(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedFilters.Statuses) != 0 {
		t.Errorf("expected no status filter for Admin, got %v", capturedFilters.Statuses)
	}
}

func TestListMaterials_Admin_NotVisibleGroup_SeesAll(t *testing.T) {
	var capturedFilters domainMaterial.ListFilters
	repo := &mockMaterialRepository{
		listFn: func(_ context.Context, _ string, f domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
			capturedFilters = f
			return nil, 0, nil
		},
	}
	uc := newListUC(repo, notVisibleGroup(), notLead())
	in := defaultListInput()
	in.CurrentUser = asAdmin(testOtherID)

	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected no error for admin on non-visible group, got %v", err)
	}
	if len(capturedFilters.Statuses) != 0 {
		t.Errorf("expected no status filter for admin, got %v", capturedFilters.Statuses)
	}
}

func TestListMaterials_PaginationMetadata(t *testing.T) {
	materials := []*domainMaterial.Material{newPublishedMaterial()}
	uc := newListUC(repoWithList(materials), visibleGroup(), isMemberNotLead())
	in := defaultListInput()
	in.Page = 1
	in.Limit = 20

	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Pagination.TotalCount != 1 {
		t.Errorf("expected totalCount=1, got %d", out.Pagination.TotalCount)
	}
	if out.Pagination.CurrentPage != 1 {
		t.Errorf("expected currentPage=1, got %d", out.Pagination.CurrentPage)
	}
	if out.Pagination.TotalPages != 1 {
		t.Errorf("expected totalPages=1, got %d", out.Pagination.TotalPages)
	}
	if out.Pagination.ItemsPerPage != 20 {
		t.Errorf("expected itemsPerPage=20, got %d", out.Pagination.ItemsPerPage)
	}
}

func TestListMaterials_AuthorPopulated(t *testing.T) {
	uc := newListUC(repoWithList([]*domainMaterial.Material{newPublishedMaterial()}), visibleGroup(), isMemberNotLead())
	out, err := uc.Execute(context.Background(), defaultListInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Materials) == 0 {
		t.Fatal("expected at least one material")
	}
	if out.Materials[0].Author == nil {
		t.Error("expected author to be populated")
	}
}

func TestListMaterials_EmptyResult_ReturnsPaginationZero(t *testing.T) {
	uc := newListUC(repoWithList(nil), visibleGroup(), isMemberNotLead())
	out, err := uc.Execute(context.Background(), defaultListInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Materials) != 0 {
		t.Errorf("expected empty materials, got %d", len(out.Materials))
	}
	if out.Pagination.TotalCount != 0 {
		t.Errorf("expected totalCount=0, got %d", out.Pagination.TotalCount)
	}
	if out.Pagination.TotalPages != 0 {
		t.Errorf("expected totalPages=0, got %d", out.Pagination.TotalPages)
	}
}

func TestListMaterials_LeadCheckError_Returns500(t *testing.T) {
	mem := &mockGroupMemberProvider{
		isLeadFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	uc := NewListMaterialsUseCase(&mockMaterialRepository{}, visibleGroup(), mem, stubAuthorProvider())
	_, err := uc.Execute(context.Background(), defaultListInput())
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}

func TestListMaterials_RepoError_Returns500(t *testing.T) {
	repo := &mockMaterialRepository{
		listFn: func(_ context.Context, _ string, _ domainMaterial.ListFilters) ([]*domainMaterial.Material, int, error) {
			return nil, 0, apperror.NewInternal()
		},
	}
	uc := newListUC(repo, visibleGroup(), isMemberNotLead())
	_, err := uc.Execute(context.Background(), defaultListInput())
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}
