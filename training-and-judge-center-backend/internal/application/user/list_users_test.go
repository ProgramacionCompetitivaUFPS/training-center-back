package user

import (
	"context"
	"errors"
	"net/http"
	"testing"

	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListUsersRepo(users []*domain.User, total int, repoErr error) *mockUserRepository {
	repo := newNoConflictRepo()
	repo.findAllFn = func(_ context.Context, _ domain.UserFilter) ([]*domain.User, int, error) {
		return users, total, repoErr
	}
	return repo
}

func TestListUsers_Success_NoFilters(t *testing.T) {
	users := []*domain.User{
		newUserWithRole("user-1", domain.RoleContestant, domain.StatusActive),
		newUserWithRole("user-2", domain.RoleCoach, domain.StatusActive),
	}
	repo := newListUsersRepo(users, 2, nil)
	uc := NewListUsersUseCase(repo)

	result, err := uc.Execute(context.Background(), ListUsersInput{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Users) != 2 {
		t.Errorf("expected 2 users, got %d", len(result.Users))
	}
	if result.TotalCount != 2 {
		t.Errorf("expected totalCount=2, got %d", result.TotalCount)
	}
	if result.Page != 1 {
		t.Errorf("expected default page=1, got %d", result.Page)
	}
	if result.Limit != 20 {
		t.Errorf("expected default limit=20, got %d", result.Limit)
	}
}

func TestListUsers_CustomPagination(t *testing.T) {
	repo := newListUsersRepo(nil, 100, nil)
	uc := NewListUsersUseCase(repo)

	result, err := uc.Execute(context.Background(), ListUsersInput{Page: 3, Limit: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Page != 3 {
		t.Errorf("expected page=3, got %d", result.Page)
	}
	if result.Limit != 10 {
		t.Errorf("expected limit=10, got %d", result.Limit)
	}
}

func TestListUsers_LimitCappedAt100(t *testing.T) {
	repo := newListUsersRepo(nil, 0, nil)
	uc := NewListUsersUseCase(repo)

	result, err := uc.Execute(context.Background(), ListUsersInput{Limit: 999})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Limit != 100 {
		t.Errorf("expected limit capped at 100, got %d", result.Limit)
	}
}

func TestListUsers_InvalidRole(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewListUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), ListUsersInput{Roles: []string{"SUPERADMIN"}})
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

func TestListUsers_InvalidStatus(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewListUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), ListUsersInput{Status: "BANNED"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestListUsers_InvalidSort(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewListUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), ListUsersInput{Sort: "password"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestListUsers_InvalidOrder(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewListUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), ListUsersInput{Order: "random"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestListUsers_InvalidSearchField(t *testing.T) {
	repo := newNoConflictRepo()
	uc := NewListUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), ListUsersInput{SearchField: "password"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "VALIDATION_ERROR" {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestListUsers_RepositoryError(t *testing.T) {
	repo := newListUsersRepo(nil, 0, errors.New("db error"))
	uc := NewListUsersUseCase(repo)

	_, err := uc.Execute(context.Background(), ListUsersInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	appErr := err.(*apperror.AppError)
	if appErr.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %q", appErr.Code)
	}
}
