package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validCreateInput(role shared.Role) CreateGroupInput {
	return CreateGroupInput{
		Name:        "Algorithms Club",
		Description: nil,
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: currentUser("u1", role),
	}
}

func newCreateGroupUC(repo *fakeRepo, memberRepo *fakeMemberRepo) *CreateGroupUseCase {
	if memberRepo == nil {
		memberRepo = &fakeMemberRepo{}
	}
	return NewCreateGroupUseCase(repo, memberRepo, &fakeTxManager{})
}

func TestCreateGroup_NonAdminNonCoachReturns403(t *testing.T) {
	uc := newCreateGroupUC(&fakeRepo{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateGroup_ValidCoachCreatesGroup(t *testing.T) {
	repo := &fakeRepo{}
	uc := newCreateGroupUC(repo, nil)

	out, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Group.Name().Value() != "Algorithms Club" {
		t.Errorf("Name = %q, want %q", out.Group.Name().Value(), "Algorithms Club")
	}
	if out.Group.JoinPolicy() != domainGroup.JoinPolicyOpen {
		t.Errorf("JoinPolicy = %v, want OPEN", out.Group.JoinPolicy())
	}
	if out.Group.Visibility() != domainGroup.VisibilityVisible {
		t.Errorf("Visibility = %v, want VISIBLE", out.Group.Visibility())
	}
	if out.Group.ID() == "" {
		t.Error("expected non-empty group ID")
	}
}

func TestCreateGroup_ValidAdminCreatesGroup(t *testing.T) {
	repo := &fakeRepo{}
	uc := newCreateGroupUC(repo, nil)

	out, err := uc.Execute(context.Background(), validCreateInput(shared.RoleAdmin))
	if err != nil {
		t.Fatalf("admin should be able to create groups, got: %v", err)
	}
	if out.Group == nil {
		t.Error("expected non-nil group")
	}
}

func TestCreateGroup_EmptyNameReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUC(&fakeRepo{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "",
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
	uc := newCreateGroupUC(&fakeRepo{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "INVALID_MODE",
		Visibility:  "VISIBLE",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
	uc := newCreateGroupUC(&fakeRepo{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "OPEN",
		Visibility:  "INVISIBLE",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
	uc := newCreateGroupUC(&fakeRepo{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "",
		JoinMode:    "NOPE",
		Visibility:  "VISIBLE",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
	repo := &fakeRepo{
		existsByNameFn: func(_ domainGroup.GroupName) (bool, error) { return true, nil },
	}
	uc := newCreateGroupUC(repo, nil)

	_, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeNameAlreadyExists {
		t.Fatalf("expected NAME_ALREADY_EXISTS, got %v", err)
	}
}

func TestCreateGroup_RepoSaveFailureReturnsInternal(t *testing.T) {
	repo := &fakeRepo{saveErr: errors.New("db failure")}
	uc := newCreateGroupUC(repo, nil)

	_, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR on save failure, got %v", err)
	}
}

func TestCreateGroup_InvalidPolicyCombinationReturnsError(t *testing.T) {
	uc := newCreateGroupUC(&fakeRepo{}, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "Secret Club",
		JoinMode:    "OPEN",
		Visibility:  "NOT_VISIBLE",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
