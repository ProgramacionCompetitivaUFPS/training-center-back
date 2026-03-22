package user

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestAdminDeactivateUser_Success(t *testing.T) {
	repo := newNoConflictRepo()
	target := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "target-1" {
			return target, nil
		}
		return nil, nil
	}
	invCalled := false
	invalidator := &mockSessionInvalidator{
		invalidateAllUserSessionsFn: func(ctx context.Context, userID string, timestamp time.Time) error {
			invCalled = true
			if userID != "target-1" {
				t.Errorf("expected target-1, got %s", userID)
			}
			return nil
		},
	}
	uc := NewAdminDeactivateUserUseCase(repo, invalidator)

	err := uc.Execute(context.Background(), "admin-1", "target-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !invCalled {
		t.Error("expected session invalidator to be called")
	}
	if target.Status != domain.StatusDeactivated {
		t.Errorf("expected status DEACTIVATED, got %s", target.Status)
	}
	if target.Email != nil {
		t.Errorf("expected email to be nil after deactivation, got %v", target.Email)
	}
	if target.DeactivatedAt == nil {
		t.Error("expected deactivatedAt to be set")
	}
	if target.UpdatedAt == nil {
		t.Error("expected updatedAt to be set")
	}
}

func TestAdminDeactivateUser_SelfDeactivation(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewAdminDeactivateUserUseCase(repo, &mockSessionInvalidator{})

	err := uc.Execute(context.Background(), "admin-1", "admin-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "CANNOT_SELF_DEACTIVATE" {
		t.Errorf("expected code CANNOT_SELF_DEACTIVATE, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", appErr.StatusCode)
	}
}

func TestAdminDeactivateUser_TargetNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewAdminDeactivateUserUseCase(repo, &mockSessionInvalidator{})

	err := uc.Execute(context.Background(), "admin-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestAdminDeactivateUser_CannotDeactivateAdmin(t *testing.T) {
	repo := newNoConflictRepo()
	adminTarget := newUserWithRole("admin-2", domain.RoleAdmin, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return adminTarget, nil
	}
	uc := NewAdminDeactivateUserUseCase(repo, &mockSessionInvalidator{})

	err := uc.Execute(context.Background(), "admin-1", "admin-2")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "CANNOT_DEACTIVATE_ADMIN" {
		t.Errorf("expected code CANNOT_DEACTIVATE_ADMIN, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", appErr.StatusCode)
	}
}

func TestAdminDeactivateUser_AlreadyDeactivated_Idempotent(t *testing.T) {
	repo := newNoConflictRepo()
	target := newUserWithRole("target-1", domain.RoleContestant, domain.StatusDeactivated)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return target, nil
	}
	updateCalled := false
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		updateCalled = true
		return nil
	}
	uc := NewAdminDeactivateUserUseCase(repo, &mockSessionInvalidator{})

	err := uc.Execute(context.Background(), "admin-1", "target-1")
	if err != nil {
		t.Fatalf("expected no error for already deactivated user, got %v", err)
	}
	if updateCalled {
		t.Error("expected Update NOT to be called for already deactivated user (idempotent)")
	}
}

func TestAdminDeactivateUser_RepositoryFindError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db error")
	}
	uc := NewAdminDeactivateUserUseCase(repo, &mockSessionInvalidator{})

	err := uc.Execute(context.Background(), "admin-1", "target-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestAdminDeactivateUser_RepositoryUpdateError(t *testing.T) {
	repo := newNoConflictRepo()
	target := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return target, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return errors.New("db update failed")
	}
	uc := NewAdminDeactivateUserUseCase(repo, &mockSessionInvalidator{})

	err := uc.Execute(context.Background(), "admin-1", "target-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}
