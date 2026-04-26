package material

import (
	"context"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGetUC(repo *mockMaterialRepository, vis *mockGroupVisibilityProvider, mem *mockGroupMemberProvider) *GetMaterial {
	return NewGetMaterial(repo, vis, mem, noopAuthorProvider())
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
	assertErrCode(t, err, ErrCodeInsufficientPerms)
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
	m := newPublishedMaterial() // belongs to testGroupID
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
	repo := repoWith(newTestMaterial()) // DRAFT
	uc := newGetUC(repo, visibleGroup(), isMemberNotLead())
	_, err := uc.Execute(context.Background(), GetMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})
	assertErrCode(t, err, domainMaterial.ErrCodeMaterialNotFound)
}

func TestGetMaterial_DraftMaterial_Lead_Returns200(t *testing.T) {
	repo := repoWith(newTestMaterial()) // DRAFT
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
	repo := repoWith(newTestMaterial()) // DRAFT
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
