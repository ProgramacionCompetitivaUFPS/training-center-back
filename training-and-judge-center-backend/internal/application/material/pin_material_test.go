package material

import (
	"context"
	"errors"
	"testing"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newPinMaterialUseCase(repo *mockMaterialRepository, group *mockGroupProvider, member *mockGroupMemberProvider) *PinMaterialUseCase {
	return NewPinMaterialUseCase(repo, group, member, stubAuthorProvider())
}

func TestPinMaterial_SuccessByAuthor(t *testing.T) {
	m := newPublishedMaterial()
	saved := false
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { saved = true; return nil },
	}
	uc := newPinMaterialUseCase(repo, groupExists(), notLead())

	out, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Material.Pinned {
		t.Error("expected pinned=true")
	}
	if out.Material.PinnedAt == nil {
		t.Error("expected pinnedAt to be set")
	}
	if !saved {
		t.Error("expected repo.Save to be called")
	}
}

func TestPinMaterial_SuccessByOtherLead(t *testing.T) {
	m := newPublishedMaterial()
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), isLead())

	out, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Material.Pinned {
		t.Error("expected pinned=true")
	}
}

func TestPinMaterial_SuccessByAdmin(t *testing.T) {
	m := newPublishedMaterial()
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), notLead())

	out, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Material.Pinned {
		t.Error("expected pinned=true")
	}
}

func TestPinMaterial_Admin_SkipsLeadCheck(t *testing.T) {
	m := newPublishedMaterial()
	leadCheckCalled := false
	member := &mockGroupMemberProvider{
		isLeadFn: func(_ context.Context, _, _ string) (bool, error) {
			leadCheckCalled = true
			return false, errors.New("should not be called")
		},
	}
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), member)

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asAdmin(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if leadCheckCalled {
		t.Error("IsLeadOfGroup must not be called for admin users")
	}
}

func TestPinMaterial_Idempotent_AlreadyPinned(t *testing.T) {
	m := newPinnedMaterial()
	originalPinnedAt := m.PinnedAt()
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), notLead())

	out, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Material.Pinned {
		t.Error("expected pinned=true")
	}
	if out.Material.PinnedAt == nil || !out.Material.PinnedAt.Equal(*originalPinnedAt) {
		t.Error("pinnedAt must not change when already pinned")
	}
}

func TestPinMaterial_CannotPinDraft(t *testing.T) {
	m := newTestMaterial()
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), notLead())

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeCannotPinDraft {
		t.Errorf("expected CANNOT_PIN_DRAFT, got %v", err)
	}
}

func TestPinMaterial_Forbidden_Member(t *testing.T) {
	m := newPublishedMaterial()
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestPinMaterial_MemberProviderError_Returns500(t *testing.T) {
	m := newPublishedMaterial()
	member := &mockGroupMemberProvider{
		isLeadFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, apperror.NewInternal()
		},
	}
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), member)

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testOtherID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from member provider failure, got %v", err)
	}
}

func TestPinMaterial_GroupProviderError_Returns500(t *testing.T) {
	uc := newPinMaterialUseCase(&mockMaterialRepository{}, groupProviderError(), notLead())

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from group provider failure, got %v", err)
	}
}

func TestPinMaterial_GroupNotFound(t *testing.T) {
	uc := newPinMaterialUseCase(&mockMaterialRepository{}, groupNotFound(), notLead())

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestPinMaterial_MaterialInOtherGroup(t *testing.T) {
	m := newPublishedMaterial()
	uc := newPinMaterialUseCase(repoWith(m), groupExists(), notLead())

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     "other-group-id",
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainMaterial.ErrCodeMaterialNotFound {
		t.Errorf("expected MATERIAL_NOT_FOUND for cross-group access, got %v", err)
	}
}

func TestPinMaterial_SaveError(t *testing.T) {
	m := newPublishedMaterial()
	repo := &mockMaterialRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainMaterial.Material, error) { return m, nil },
		saveFn:     func(_ context.Context, _ *domainMaterial.Material) error { return apperror.NewInternal() },
	}
	uc := newPinMaterialUseCase(repo, groupExists(), notLead())

	_, err := uc.Execute(context.Background(), PinMaterialInput{
		CurrentUser: asCoach(testAuthorID),
		GroupID:     testGroupID,
		MaterialID:  testMaterialID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR from save failure, got %v", err)
	}
}
