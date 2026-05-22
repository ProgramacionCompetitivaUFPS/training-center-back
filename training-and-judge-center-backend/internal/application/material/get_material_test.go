package material

import (
	"context"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGetUC(repo *mockMaterialRepository, vis *mockGroupVisibilityProvider, mem *mockGroupMemberProvider) *GetMaterialUseCase {
	return NewGetMaterialUseCase(repo, vis, mem, stubAuthorProvider())
}

func TestGetMaterial_GroupNotFound_Returns404(t *testing.T) {
	uc := newGetUC(&mockMaterialRepository{}, groupVisibilityNotFound(), notLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, ErrCodeGroupNotFound)
}

func TestGetMaterial_NotVisibleGroup_NonMember_Returns403(t *testing.T) {
	uc := newGetUC(&mockMaterialRepository{}, notVisibleGroup(), notLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, ErrCodeInsufficientPermissions)
}

func TestGetMaterial_NotVisibleGroup_Admin_CanAccess(t *testing.T) {
	repo := repoWith(newPublishedMaterial())
	uc := newGetUC(repo, notVisibleGroup(), notLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGetMaterial_MaterialNotFound_Returns404(t *testing.T) {
	uc := newGetUC(&mockMaterialRepository{}, visibleGroup(), isMemberNotLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, domainMaterial.ErrCodeMaterialNotFound)
}

func TestGetMaterial_MaterialBelongsToDifferentGroup_Returns404(t *testing.T) {
	m := newPublishedMaterial()
	repo := repoWith(m)
	uc := newGetUC(repo, visibleGroup(), isMemberNotLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "other-group-id",
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, domainMaterial.ErrCodeMaterialNotFound)
}

func TestGetMaterial_DraftMaterial_Member_Returns404(t *testing.T) {
	repo := repoWith(newTestMaterial())
	uc := newGetUC(repo, visibleGroup(), isMemberNotLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, domainMaterial.ErrCodeMaterialNotFound)
}

func TestGetMaterial_DraftMaterial_Lead_Returns200(t *testing.T) {
	repo := repoWith(newTestMaterial())
	uc := newGetUC(repo, visibleGroup(), isLead())
	out, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Material.Status != "DRAFT" {
		t.Errorf("expected DRAFT, got %s", out.Material.Status)
	}
}

func TestGetMaterial_DraftMaterial_Admin_Returns200(t *testing.T) {
	repo := repoWith(newTestMaterial())
	uc := newGetUC(repo, visibleGroup(), notLead())
	out, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Material.Status != "DRAFT" {
		t.Errorf("expected DRAFT, got %s", out.Material.Status)
	}
}

func TestGetMaterial_PublishedMaterial_Member_Returns200(t *testing.T) {
	repo := repoWith(newPublishedMaterial())
	uc := newGetUC(repo, visibleGroup(), isMemberNotLead())
	out, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asContestant(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Material.Status != "PUBLISHED" {
		t.Errorf("expected PUBLISHED, got %s", out.Material.Status)
	}
}

func TestGetMaterial_AuthorPopulated(t *testing.T) {
	repo := repoWith(newPublishedMaterial())
	uc := newGetUC(repo, visibleGroup(), isMemberNotLead())
	out, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Material.Author == nil {
		t.Fatal("expected author to be populated")
	}
	if out.Material.Author.Nickname == "" {
		t.Error("expected non-empty author nickname")
	}
}

// assertErrCode is a shared helper.
func assertErrCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", code)
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Code != code {
		t.Errorf("expected error code %q, got %q", code, appErr.Code)
	}
}

func TestGetMaterial_AuthorProviderError_Returns500(t *testing.T) {
	repo := repoWith(newPublishedMaterial())
	authorProvider := &mockAuthorProvider{
		getDisplaysFn: func(_ context.Context, _ []string) (map[string]*AuthorDisplay, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := NewGetMaterialUseCase(repo, visibleGroup(), isMemberNotLead(), authorProvider)

	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err == nil {
		t.Fatal("expected error on author provider failure, got nil")
	}
}

func TestListMaterials_AuthorProviderError_Returns500(t *testing.T) {
	authorProvider := &mockAuthorProvider{
		getDisplaysFn: func(_ context.Context, _ []string) (map[string]*AuthorDisplay, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := NewListMaterialsUseCase(repoWithList([]*domainMaterial.Material{newPublishedMaterial()}), visibleGroup(), isMemberNotLead(), authorProvider)

	_, err := uc.Execute(context.Background(), defaultListInput())

	if err == nil {
		t.Fatal("expected error on author provider failure, got nil")
	}
}

func TestGetMaterial_GroupVisibilityError_Returns500(t *testing.T) {
	vis := &mockGroupVisibilityProvider{
		findVisibilityFn: func(_ context.Context, _ string) (GroupVisibility, bool, error) {
			return "", false, apperror.NewInternal()
		},
	}
	uc := newGetUC(&mockMaterialRepository{}, vis, notLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}

func TestListMaterials_GroupVisibilityError_Returns500(t *testing.T) {
	vis := &mockGroupVisibilityProvider{
		findVisibilityFn: func(_ context.Context, _ string) (GroupVisibility, bool, error) {
			return "", false, apperror.NewInternal()
		},
	}
	uc := newListUC(&mockMaterialRepository{}, vis, notLead())
	_, err := uc.Execute(context.Background(), defaultListInput())
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}

func TestGetMaterial_MemberCheckError_Returns500(t *testing.T) {
	mem := &mockGroupMemberProvider{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newGetUC(&mockMaterialRepository{}, notVisibleGroup(), mem)
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}

func TestListMaterials_MemberCheckError_Returns500(t *testing.T) {
	mem := &mockGroupMemberProvider{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newListUC(&mockMaterialRepository{}, notVisibleGroup(), mem)
	_, err := uc.Execute(context.Background(), defaultListInput())
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}

func TestGetMaterial_LeadCheckError_Returns500(t *testing.T) {
	mem := &mockGroupMemberProvider{
		isLeadFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) {
			return true, nil
		},
	}
	repo := repoWith(newTestMaterial())
	uc := NewGetMaterialUseCase(repo, visibleGroup(), mem, stubAuthorProvider())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, apperror.ErrCodeInternalError)
}
