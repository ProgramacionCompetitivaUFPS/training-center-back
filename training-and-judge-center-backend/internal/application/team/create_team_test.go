package team

import (
	"context"
	"testing"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestCreateTeam_ValidContestantCreatesTeam(t *testing.T) {
	uc := newCreateTeamUseCase(&mockTeamRepository{}, nil)

	out, err := uc.Execute(context.Background(), validInput(asContestant("u1")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Alpha Team" {
		t.Errorf("Name = %q, want %q", out.Name, "Alpha Team")
	}
	if out.ID == "" {
		t.Error("expected non-empty team ID")
	}
	if out.CreatedBy != "u1" {
		t.Errorf("CreatedBy = %q, want %q", out.CreatedBy, "u1")
	}
	if len(out.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(out.Members))
	}
	if out.Members[0].UserID != "u1" {
		t.Errorf("member UserID = %q, want %q", out.Members[0].UserID, "u1")
	}
	if out.Members[0].Nickname != "testuser" {
		t.Errorf("member Nickname = %q, want %q", out.Members[0].Nickname, "testuser")
	}
}

func TestCreateTeam_EmptyNameReturnsValidationError(t *testing.T) {
	uc := newCreateTeamUseCase(&mockTeamRepository{}, nil)

	_, err := uc.Execute(context.Background(), CreateTeamInput{
		Name:        "",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "name" {
		t.Errorf("expected field error on 'name', got %+v", ae.Details)
	}
}

func TestCreateTeam_DuplicateNameReturnsConflict(t *testing.T) {
	repo := &mockTeamRepository{
		existsByNameFn: func(_ domainTeam.TeamName) (bool, error) { return true, nil },
	}
	uc := newCreateTeamUseCase(repo, nil)

	_, err := uc.Execute(context.Background(), validInput(asContestant("u1")))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainTeam.ErrCodeTeamNameExists {
		t.Fatalf("expected TEAM_NAME_EXISTS, got %v", err)
	}
}

func TestCreateTeam_RepoSaveFailureReturnsInternal(t *testing.T) {
	repo := &mockTeamRepository{saveErr: apperror.NewInternal()}
	uc := newCreateTeamUseCase(repo, nil)

	_, err := uc.Execute(context.Background(), validInput(asContestant("u1")))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestCreateTeam_CreatorIsFirstMember(t *testing.T) {
	uc := newCreateTeamUseCase(&mockTeamRepository{}, nil)

	out, err := uc.Execute(context.Background(), validInput(asAdmin("admin-1")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Members[0].UserID != "admin-1" {
		t.Errorf("expected creator as first member, got %q", out.Members[0].UserID)
	}
}

func TestCreateTeam_NicknameFromProviderIncluded(t *testing.T) {
	uc := NewCreateTeamUseCase(
		&mockTeamRepository{},
		&mockMemberRepository{},
		&mockUserProvider{displayFn: func(_ string) (*UserDisplay, error) {
			return &UserDisplay{Nickname: "champ"}, nil
		}},
		&mockTxManager{},
	)

	out, err := uc.Execute(context.Background(), validInput(asContestant("u1")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Members[0].Nickname != "champ" {
		t.Errorf("Nickname = %q, want %q", out.Members[0].Nickname, "champ")
	}
}
