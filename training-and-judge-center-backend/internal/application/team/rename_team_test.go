package team

import (
	"errors"
	"testing"

	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// teamWithMembers returns a repository of one team ("Alpha") and a member repository where only the
// given users belong to it.
func teamWithMembers(memberIDs ...string) (*mockTeamRepository, *mockMemberRepository) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(id string) (*domainTeam.Team, error) {
			if id != testTeamID {
				return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
			}
			return makeTeam(testTeamID, "Alpha", memberA, testNow), nil
		},
	}
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(teamID string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			for _, id := range memberIDs {
				if id == userID.Value() {
					return makeMember("m-"+id, teamID, id, testNow), nil
				}
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
	}
	return teamRepo, memberRepo
}

func TestRenameTeam_AnyMemberCanRename(t *testing.T) {
	// Arrange: memberB did not create the team
	teamRepo, memberRepo := teamWithMembers(memberA, memberB)
	var saved string
	teamRepo.updateNameFn = func(team *domainTeam.Team) error {
		saved = team.Name().Value()
		return nil
	}
	uc := NewRenameTeamUseCase(teamRepo, memberRepo)

	// Act
	out, err := uc.Execute(ctx(), RenameTeamInput{CurrentUser: asContestant(memberB), TeamID: testTeamID, Name: "  Beta Squad  "})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Beta Squad" || saved != "Beta Squad" {
		t.Errorf("expected the trimmed name to be saved and returned, got out=%q saved=%q", out.Name, saved)
	}
	if out.ID != testTeamID {
		t.Errorf("expected team %s, got %s", testTeamID, out.ID)
	}
}

func TestRenameTeam_ChangingOnlyTheCaseIsAllowed(t *testing.T) {
	// Arrange: "ALPHA" differs from "Alpha" only by case, which the unique index treats as the same
	// name; renaming must not run a separate "name exists" check that would find the team itself.
	teamRepo, memberRepo := teamWithMembers(memberA)
	teamRepo.existsByNameFn = func(_ domainTeam.TeamName) (bool, error) { return true, nil }
	called := false
	teamRepo.updateNameFn = func(_ *domainTeam.Team) error { called = true; return nil }
	uc := NewRenameTeamUseCase(teamRepo, memberRepo)

	// Act
	out, err := uc.Execute(ctx(), RenameTeamInput{CurrentUser: asContestant(memberA), TeamID: testTeamID, Name: "ALPHA"})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called || out.Name != "ALPHA" {
		t.Errorf("expected the team to be renamed to ALPHA, got called=%v name=%q", called, out.Name)
	}
}

func TestRenameTeam_NonMemberIsForbidden(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA)
	teamRepo.updateNameFn = func(_ *domainTeam.Team) error {
		t.Error("a non-member must not be able to rename the team")
		return nil
	}
	uc := NewRenameTeamUseCase(teamRepo, memberRepo)

	_, err := uc.Execute(ctx(), RenameTeamInput{CurrentUser: asContestant(strangerID), TeamID: testTeamID, Name: "Hijacked"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden || appErr.Code != domainTeam.ErrCodeNotTeamMember {
		t.Fatalf("expected forbidden NOT_TEAM_MEMBER, got %v", err)
	}
}

func TestRenameTeam_UnknownTeamReturnsNotFound(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA)
	uc := NewRenameTeamUseCase(teamRepo, memberRepo)

	_, err := uc.Execute(ctx(), RenameTeamInput{CurrentUser: asContestant(memberA), TeamID: "no-such-team", Name: "Gamma"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainTeam.ErrCodeTeamNotFound {
		t.Fatalf("expected TEAM_NOT_FOUND, got %v", err)
	}
}

func TestRenameTeam_InvalidNameIsRejectedBeforeSaving(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA)
	teamRepo.updateNameFn = func(_ *domainTeam.Team) error {
		t.Error("an invalid name must not be saved")
		return nil
	}
	uc := NewRenameTeamUseCase(teamRepo, memberRepo)

	_, err := uc.Execute(ctx(), RenameTeamInput{CurrentUser: asContestant(memberA), TeamID: testTeamID, Name: "   "})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Fatalf("expected a validation error, got %v", err)
	}
}

func TestRenameTeam_NameTakenByAnotherTeamReturnsConflict(t *testing.T) {
	teamRepo, memberRepo := teamWithMembers(memberA)
	teamRepo.updateNameFn = func(_ *domainTeam.Team) error {
		return apperror.NewConflict(domainTeam.ErrCodeTeamNameExists, "A team with this name already exists")
	}
	uc := NewRenameTeamUseCase(teamRepo, memberRepo)

	_, err := uc.Execute(ctx(), RenameTeamInput{CurrentUser: asContestant(memberA), TeamID: testTeamID, Name: "Taken"})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindConflict || appErr.Code != domainTeam.ErrCodeTeamNameExists {
		t.Fatalf("expected conflict TEAM_NAME_EXISTS, got %v", err)
	}
}
