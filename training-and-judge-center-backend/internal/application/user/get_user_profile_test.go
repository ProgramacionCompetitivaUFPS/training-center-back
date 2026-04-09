package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetMyProfile_Success(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	result, err := uc.GetMyProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.IsFullProfile {
		t.Error("expected IsFullProfile to be true for own profile")
	}
	if result.User.ID != "user-1" {
		t.Errorf("expected user ID %q, got %q", "user-1", result.User.ID)
	}
	if result.User.Email == nil || *result.User.Email != "user-1@example.com" {
		t.Errorf("expected email %q, got %v", "user-1@example.com", result.User.Email)
	}
}

func TestGetMyProfile_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetMyProfile(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", appErr.StatusCode)
	}
}

func TestGetMyProfile_RepositoryError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db connection lost")
	}
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetMyProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestGetByNickname_PublicProfile(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target", domain.RoleCoach, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	result, err := uc.GetUserByNickname(context.Background(), "requester-1", domain.RoleContestant, "target")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.IsFullProfile {
		t.Error("expected IsFullProfile to be false for non-admin viewing other user")
	}
	if result.User.Name != "User target" {
		t.Errorf("expected name %q, got %q", "User target", result.User.Name)
	}
}

func TestGetByNickname_AdminViewsAll(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target", domain.RoleContestant, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	result, err := uc.GetUserByNickname(context.Background(), "admin-1", domain.RoleAdmin, "target")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.IsFullProfile {
		t.Error("expected IsFullProfile to be true for admin viewing other user")
	}
}

func TestGetByNickname_SelfViaNickname(t *testing.T) {
	repo := newNoConflictRepo()
	selfUser := newUserWithRole("self-user", domain.RoleContestant, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return selfUser, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	result, err := uc.GetUserByNickname(context.Background(), "self-user", domain.RoleContestant, "self-user")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.IsFullProfile {
		t.Error("expected IsFullProfile to be true when viewing own profile via nickname")
	}
}

func TestGetByNickname_NonAdminViewsAdmin(t *testing.T) {
	repo := newNoConflictRepo()
	adminUser := newUserWithRole("admin-target", domain.RoleAdmin, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return adminUser, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetUserByNickname(context.Background(), "requester-1", domain.RoleContestant, "admin-target")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "ADMIN_PROFILE_RESTRICTED" {
		t.Errorf("expected code ADMIN_PROFILE_RESTRICTED, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", appErr.StatusCode)
	}
}

func TestGetByNickname_DeactivatedUser(t *testing.T) {
	repo := newNoConflictRepo()
	deactivatedUser := newUserWithRole("deactivated", domain.RoleContestant, domain.StatusDeactivated)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetUserByNickname(context.Background(), "requester-1", domain.RoleContestant, "deactivated")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestGetByNickname_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetUserByNickname(context.Background(), "requester-1", domain.RoleContestant, "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestGetByNickname_InvalidNickname(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetUserByNickname(context.Background(), "requester-1", domain.RoleContestant, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", appErr.StatusCode)
	}
}

func TestGetByNickname_AdminViewsDeactivated(t *testing.T) {
	repo := newNoConflictRepo()
	deactivatedUser := newUserWithRole("deactivated", domain.RoleContestant, domain.StatusDeactivated)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewGetUserProfileUseCase(repo)

	_, err := uc.GetUserByNickname(context.Background(), "admin-1", domain.RoleAdmin, "deactivated")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q (deactivated users return 404 even for admins)", appErr.Code)
	}
}
