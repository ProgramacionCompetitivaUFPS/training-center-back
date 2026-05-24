package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validCreateInput(role shared.Role) CreateGroupInput {
	cu := asContestant("u1")
	switch role {
	case shared.RoleAdmin:
		cu = asAdmin("u1")
	case shared.RoleCoach:
		cu = asCoach("u1")
	}
	return CreateGroupInput{
		Name:        "Algorithms Club",
		Description: nil,
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: cu,
	}
}

func newCreateGroupUseCase(repo *mockGroupRepository, memberRepo *mockMemberRepository) *CreateGroupUseCase {
	if memberRepo == nil {
		memberRepo = &mockMemberRepository{}
	}
	return NewCreateGroupUseCase(repo, memberRepo, &mockTransactionManager{})
}

func TestCreateGroup_NonAdminNonCoachReturns403(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateGroup_ValidCoachCreatesGroup(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := newCreateGroupUseCase(repo, nil)

	out, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Algorithms Club" {
		t.Errorf("Name = %q, want %q", out.Name, "Algorithms Club")
	}
	if out.JoinPolicy != "OPEN" {
		t.Errorf("JoinPolicy = %q, want OPEN", out.JoinPolicy)
	}
	if out.Visibility != "VISIBLE" {
		t.Errorf("Visibility = %q, want VISIBLE", out.Visibility)
	}
	if out.ID == "" {
		t.Error("expected non-empty group ID")
	}
}

func TestCreateGroup_ValidAdminCreatesGroup(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := newCreateGroupUseCase(repo, nil)

	out, err := uc.Execute(context.Background(), validCreateInput(shared.RoleAdmin))
	if err != nil {
		t.Fatalf("admin should be able to create groups, got: %v", err)
	}
	if out.ID == "" {
		t.Error("expected non-empty group ID")
	}
}

func TestCreateGroup_EmptyNameReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "",
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "name" {
		t.Errorf("expected field error on 'name', got %+v", ae.Details)
	}
}

func TestCreateGroup_InvalidJoinModeReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "INVALID_MODE",
		Visibility:  "VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "joinPolicy" {
		t.Errorf("expected field error on 'joinPolicy', got %+v", ae.Details)
	}
}

func TestCreateGroup_InvalidVisibilityReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "OPEN",
		Visibility:  "INVISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "visibility" {
		t.Errorf("expected field error on 'visibility', got %+v", ae.Details)
	}
}

func TestCreateGroup_MultipleInvalidFieldsAccumulated(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "",
		JoinMode:    "NOPE",
		Visibility:  "VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) != 2 {
		t.Errorf("expected 2 field errors (name + joinPolicy), got %d: %+v", len(ae.Details), ae.Details)
	}
}

func TestCreateGroup_DuplicateNameReturnsConflict(t *testing.T) {
	repo := &mockGroupRepository{
		existsByNameFn: func(_ domainGroup.GroupName) (bool, error) { return true, nil },
	}
	uc := newCreateGroupUseCase(repo, nil)

	_, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeNameAlreadyExists {
		t.Fatalf("expected NAME_ALREADY_EXISTS, got %v", err)
	}
}

func TestCreateGroup_RepoSaveFailureReturnsInternal(t *testing.T) {
	repo := &mockGroupRepository{saveErr: apperror.NewInternal()}
	uc := newCreateGroupUseCase(repo, nil)

	_, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR on save failure, got %v", err)
	}
}

func TestCreateGroup_InvalidPolicyCombinationReturnsError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "Secret Club",
		JoinMode:    "OPEN",
		Visibility:  "NOT_VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if ae.Code != apperror.ErrCodeValidationError {
		t.Errorf("Code = %q, want VALIDATION_ERROR", ae.Code)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "joinPolicy" {
		t.Errorf("expected joinPolicy field error, got %+v", ae.Details)
	}
}
