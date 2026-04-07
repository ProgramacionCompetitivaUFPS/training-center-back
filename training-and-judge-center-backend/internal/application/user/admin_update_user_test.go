package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestAdminUpdateUser_Success_AllFields(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, id string) (*domain.User, error) {
		if id == "target-1" {
			return targetUser, nil
		}
		return nil, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	newName := "Updated Name"
	newNick := "updatednick"
	newInst := "New University"
	newEmail := "updated@example.com"
	newRole := "COACH"

	result, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID:    "target-1",
		Name:        &newName,
		Nickname:    &newNick,
		Institution: &newInst,
		Email:       &newEmail,
		Role:        &newRole,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", result.Name)
	}
	if result.Nickname != "updatednick" {
		t.Errorf("expected nickname %q, got %q", "updatednick", result.Nickname)
	}
	if result.Email == nil || *result.Email != "updated@example.com" {
		t.Errorf("expected email %q, got %v", "updated@example.com", result.Email)
	}
	if result.Role != "COACH" {
		t.Errorf("expected role %q, got %q", "COACH", result.Role)
	}
	if result.UpdatedAt == nil {
		t.Error("expected updatedAt to be set")
	}
}

func TestAdminUpdateUser_Success_PartialUpdate(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	newName := "Only Name"
	result, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Name != "Only Name" {
		t.Errorf("expected name %q, got %q", "Only Name", result.Name)
	}
	if result.Role != "CONTESTANT" {
		t.Errorf("expected role unchanged, got %q", result.Role)
	}
}

func TestAdminUpdateUser_UserNotFound(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewAdminUpdateUserUseCase(repo)

	newName := "Some Name"
	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "nonexistent",
		Name: &newName,
	})
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

func TestAdminUpdateUser_EmptyPayload(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewAdminUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{TargetID: "target-1"})
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
}

func TestAdminUpdateUser_CannotAssignAdminRole(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	adminRole := "ADMIN"
	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Role: &adminRole,
	})
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
	if len(appErr.Details) == 0 || appErr.Details[0].Message != "Cannot assign ADMIN role through this endpoint" {
		t.Errorf("expected ADMIN role restriction message, got %v", appErr.Details)
	}
}

func TestAdminUpdateUser_EmailAlreadyExists(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return domain.ErrEmailConflict
	}
	uc := NewAdminUpdateUserUseCase(repo)

	takenEmail := "taken@example.com"
	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Email: &takenEmail,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "EMAIL_ALREADY_EXISTS" {
		t.Errorf("expected code EMAIL_ALREADY_EXISTS, got %q", appErr.Code)
	}
}

func TestAdminUpdateUser_RepositoryFindError(t *testing.T) {
	repo := newNoConflictRepo()
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, errors.New("db connection lost")
	}
	uc := NewAdminUpdateUserUseCase(repo)

	newName := "Some Name"
	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Name: &newName,
	})
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

func TestAdminUpdateUser_RepositoryUpdateError(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	repo.updateFn = func(_ context.Context, _ *domain.User) error {
		return errors.New("db update failed")
	}
	uc := NewAdminUpdateUserUseCase(repo)

	newName := "New Name"
	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Name: &newName,
	})
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

func TestAdminUpdateUser_InvalidRole(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	badRole := "SUPERUSER"
	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Role:     &badRole,
	})
	if err == nil {
		t.Fatal("expected error for invalid role, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %q", appErr.Code)
	}
	if len(appErr.Details) == 0 || appErr.Details[0].Field != "role" {
		t.Errorf("expected field error on role, got %v", appErr.Details)
	}
}

func TestAdminUpdateUser_Success_CityAndCountry(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	result, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		City:     strPtr("Guayaquil"),
		Country:  strPtr("Ecuador"),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.City != "Guayaquil" {
		t.Errorf("expected city %q, got %q", "Guayaquil", result.City)
	}
	if result.Country != "Ecuador" {
		t.Errorf("expected country %q, got %q", "Ecuador", result.Country)
	}
}

func TestAdminUpdateUser_EmptyCityValidation(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		City:     strPtr(""),
	})
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
}

func TestAdminUpdateUser_EmptyCountryValidation(t *testing.T) {
	repo := newNoConflictRepo()
	targetUser := newUserWithRole("target-1", domain.RoleContestant, domain.StatusActive)
	repo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return targetUser, nil
	}
	uc := NewAdminUpdateUserUseCase(repo)

	_, err := uc.Execute(context.Background(), AdminUpdateUserInput{
		TargetID: "target-1",
		Country:  strPtr(""),
	})
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
}
