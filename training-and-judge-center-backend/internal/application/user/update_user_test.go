package user

import (
	"context"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func strPtr(s string) *string { return &s }

func TestUpdateUser_Success_AllFields(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID:      "user-1",
		Name:        strPtr("Updated Name"),
		Nickname:    strPtr("updatednick"),
		Institution: strPtr("New University"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", result.User.Name)
	}
	if result.User.Nickname != "updatednick" {
		t.Errorf("expected nickname %q, got %q", "updatednick", result.User.Nickname)
	}
	if result.User.Institution != "New University" {
		t.Errorf("expected institution %q, got %q", "New University", result.User.Institution)
	}
	if result.User.UpdatedAt == nil {
		t.Error("expected updatedAt to be set")
	}
}

func TestUpdateUser_Success_PartialUpdate(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "user-1" {
			return activeUser, nil
		}
		return nil, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID: "user-1",
		Name: strPtr("Only Name Changed"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.Name != "Only Name Changed" {
		t.Errorf("expected name %q, got %q", "Only Name Changed", result.User.Name)
	}
	if result.User.Nickname != "user-1" {
		t.Errorf("expected original nickname %q, got %q", "user-1", result.User.Nickname)
	}
}

func TestUpdateUser_NoFieldsProvided(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{UserID: "user-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindValidation {
		t.Errorf("expected kind VALIDATION, got %s", appErr.Kind)
	}
}

func TestUpdateUser_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID: "nonexistent",
		Name: strPtr("New Name"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeUserNotFound {
		t.Errorf("expected code NOT_FOUND, got %q", appErr.Code)
	}
}

func TestUpdateUser_EmptyNameValidation(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID: "user-1",
		Name: strPtr(""),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateUser_NicknameAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return apperror.NewConflict(domain.ErrCodeNicknameConflict, "nickname already in use")
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID:   "user-1",
		Nickname: strPtr("taken-nick"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeNicknameConflict {
		t.Errorf("expected code %q, got %q", domain.ErrCodeNicknameConflict, appErr.Code)
	}
}

func TestUpdateUser_SameNicknameNoConflict(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID:   "user-1",
		Nickname: strPtr("user-1"),
	})
	if err != nil {
		t.Fatalf("expected no error when nickname unchanged, got %v", err)
	}
	if result.User.Nickname != "user-1" {
		t.Errorf("expected nickname %q, got %q", "user-1", result.User.Nickname)
	}
}

func TestUpdateUser_RepositoryUpdateError(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return apperror.NewInternal()
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID: "user-1",
		Name: strPtr("New Name"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected code INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateUser_NicknameLowercased(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID:   "user-1",
		Nickname: strPtr("MyNewNick"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.Nickname != "mynewnick" {
		t.Errorf("expected nickname to be lowercased %q, got %q", "mynewnick", result.User.Nickname)
	}
}

func TestUpdateUser_Success_CityAndCountry(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID:  "user-1",
		City:    strPtr("Quito"),
		Country: strPtr("Ecuador"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.User.City != "Quito" {
		t.Errorf("expected city %q, got %q", "Quito", result.User.City)
	}
	if result.User.Country != "Ecuador" {
		t.Errorf("expected country %q, got %q", "Ecuador", result.User.Country)
	}
}

func TestUpdateUser_EmptyCityValidation(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID: "user-1",
		City:   strPtr(""),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateUser_EmptyCountryValidation(t *testing.T) {
	repo := newNoConflictRepo()
	activeUser := newUserWithRole("user-1", shared.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return activeUser, nil
	}
	uc := NewUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateUserInput{
		UserID:  "user-1",
		Country: strPtr(""),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
}
