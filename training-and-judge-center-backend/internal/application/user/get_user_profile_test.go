package user

import (
	"context"
	"errors"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetMyProfile_Success(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewGetMyProfileUseCase(repo)

	result, err := uc.Execute(context.Background(), GetMyProfileInput{UserID: "user-1"})
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
	uc := NewGetMyProfileUseCase(repo)

	_, err := uc.Execute(context.Background(), GetMyProfileInput{UserID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeUserNotFound {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindNotFound {
		t.Errorf("expected kind NOT_FOUND, got %s", appErr.Kind)
	}
}

func TestGetMyProfile_RepositoryError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db connection lost")
	}
	uc := NewGetMyProfileUseCase(repo)

	_, err := uc.Execute(context.Background(), GetMyProfileInput{UserID: "user-1"})
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
	targetUser := newUserWithRole("target", shared.RoleCoach, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewGetUserByNicknameUseCase(repo)

	result, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "requester-1", RequesterRole: shared.RoleContestant, Nickname: "target"})
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
	targetUser := newUserWithRole("target", shared.RoleContestant, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewGetUserByNicknameUseCase(repo)

	result, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "admin-1", RequesterRole: shared.RoleAdmin, Nickname: "target"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.IsFullProfile {
		t.Error("expected IsFullProfile to be true for admin viewing other user")
	}
}

func TestGetByNickname_SelfViaNickname(t *testing.T) {
	repo := newNoConflictRepo()
	selfUser := newUserWithRole("self-user", shared.RoleContestant, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return selfUser, nil
	}
	uc := NewGetUserByNicknameUseCase(repo)

	result, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "self-user", RequesterRole: shared.RoleContestant, Nickname: "self-user"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.IsFullProfile {
		t.Error("expected IsFullProfile to be true when viewing own profile via nickname")
	}
}

func TestGetByNickname_NonAdminViewsAdmin(t *testing.T) {
	repo := newNoConflictRepo()
	adminUser := newUserWithRole("admin-target", shared.RoleAdmin, domain.StatusActive)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return adminUser, nil
	}
	uc := NewGetUserByNicknameUseCase(repo)

	_, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "requester-1", RequesterRole: shared.RoleContestant, Nickname: "admin-target"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeAdminProfileRestricted {
		t.Errorf("expected code ADMIN_PROFILE_RESTRICTED, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", appErr.Kind)
	}
}

func TestGetByNickname_DeactivatedUser(t *testing.T) {
	repo := newNoConflictRepo()
	deactivatedUser := newUserWithRole("deactivated", shared.RoleContestant, domain.StatusDeactivated)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewGetUserByNicknameUseCase(repo)

	_, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "requester-1", RequesterRole: shared.RoleContestant, Nickname: "deactivated"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeUserNotFound {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestGetByNickname_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewGetUserByNicknameUseCase(repo)

	_, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "requester-1", RequesterRole: shared.RoleContestant, Nickname: "nonexistent"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeUserNotFound {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestGetByNickname_InvalidNickname(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewGetUserByNicknameUseCase(repo)

	_, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "requester-1", RequesterRole: shared.RoleContestant, Nickname: ""})
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
	if appErr.Kind != apperror.KindValidation {
		t.Errorf("expected kind VALIDATION, got %s", appErr.Kind)
	}
}

func TestGetByNickname_AdminViewsDeactivated(t *testing.T) {
	repo := newNoConflictRepo()
	deactivatedUser := newUserWithRole("deactivated", shared.RoleContestant, domain.StatusDeactivated)
	repo.findByNicknameFn = func(_ context.Context, _ domain.Nickname) (*domain.User, error) {
		return deactivatedUser, nil
	}
	uc := NewGetUserByNicknameUseCase(repo)

	_, err := uc.Execute(context.Background(), GetUserByNicknameInput{RequesterID: "admin-1", RequesterRole: shared.RoleAdmin, Nickname: "deactivated"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeUserNotFound {
		t.Errorf("expected code NOT_FOUND, got %q (deactivated users return 404 even for admins)", appErr.Code)
	}
}
