package team

import (
	"testing"
	"time"

	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newInviteUseCase(teamRepo *mockTeamRepository, memberRepo *mockMemberRepository, invRepo *mockInvitationRepository, up *mockUserProvider) *InviteToTeamUseCase {
	return NewInviteToTeamUseCase(teamRepo, memberRepo, invRepo, up)
}

func TestInviteToTeam_RequesterNotMemberReturns403(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeTeam("t1", "Alpha", "creator", time.Now()), nil
		},
	}
	uc := newInviteUseCase(teamRepo, &mockMemberRepository{}, &mockInvitationRepository{}, &mockUserProvider{})
	_, err := uc.Execute(ctx(), InviteToTeamInput{
		CurrentUser: asContestant("requester"),
		TeamID:      "t1",
		Nickname:    "invitee",
	}, time.Now())

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected KindForbidden, got %v", err)
	}
}

func TestInviteToTeam_InviteeNotFoundReturns404(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeTeam("t1", "Alpha", "requester", time.Now()), nil
		},
	}
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			if userID.Value() == "requester" {
				return makeMember("m1", "t1", "requester", time.Now()), nil
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
	}
	up := &mockUserProvider{
		findByNicknameFn: func(_ string) (*UserDisplay, error) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeUserNotFound, "user not found")
		},
	}
	uc := newInviteUseCase(teamRepo, memberRepo, &mockInvitationRepository{}, up)
	_, err := uc.Execute(ctx(), InviteToTeamInput{
		CurrentUser: asContestant("requester"),
		TeamID:      "t1",
		Nickname:    "ghost",
	}, time.Now())

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected KindNotFound, got %v", err)
	}
}

func TestInviteToTeam_InviteeAlreadyMemberReturns409(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeTeam("t1", "Alpha", "requester", time.Now()), nil
		},
	}
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	up := &mockUserProvider{
		findByNicknameFn: func(_ string) (*UserDisplay, error) {
			return &UserDisplay{ID: "invitee", Nickname: "invitee_nick"}, nil
		},
	}
	uc := newInviteUseCase(teamRepo, memberRepo, &mockInvitationRepository{}, up)
	_, err := uc.Execute(ctx(), InviteToTeamInput{
		CurrentUser: asContestant("requester"),
		TeamID:      "t1",
		Nickname:    "invitee_nick",
	}, time.Now())

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindConflict {
		t.Errorf("expected KindConflict, got %v", err)
	}
	if ae.Code != domainTeam.ErrCodeMemberAlreadyOnTeam {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeMemberAlreadyOnTeam, ae.Code)
	}
}

func TestInviteToTeam_AlreadyInvitedReturns409(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeTeam("t1", "Alpha", "requester", time.Now()), nil
		},
	}
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			if userID.Value() == "requester" {
				return makeMember("m1", "t1", "requester", time.Now()), nil
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
	}
	up := &mockUserProvider{
		findByNicknameFn: func(_ string) (*UserDisplay, error) {
			return &UserDisplay{ID: "invitee", Nickname: "invitee_nick"}, nil
		},
	}
	invRepo := &mockInvitationRepository{
		findByTeamAndInviteeFn: func(_ string, _ domainShared.UserID) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1", domainShared.RestoreUserID("invitee"), domainShared.RestoreUserID("requester"), time.Now()), nil
		},
	}
	uc := newInviteUseCase(teamRepo, memberRepo, invRepo, up)
	_, err := uc.Execute(ctx(), InviteToTeamInput{
		CurrentUser: asContestant("requester"),
		TeamID:      "t1",
		Nickname:    "invitee_nick",
	}, time.Now())

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindConflict {
		t.Errorf("expected KindConflict, got %v", err)
	}
	if ae.Code != domainTeam.ErrCodeAlreadyInvited {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeAlreadyInvited, ae.Code)
	}
}

func TestInviteToTeam_SuccessReturns201(t *testing.T) {
	teamRepo := &mockTeamRepository{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeTeam("t1", "Alpha", "requester", time.Now()), nil
		},
	}
	memberRepo := &mockMemberRepository{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			if userID.Value() == "requester" {
				return makeMember("m1", "t1", "requester", time.Now()), nil
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
	}
	up := &mockUserProvider{
		findByNicknameFn: func(_ string) (*UserDisplay, error) {
			return &UserDisplay{ID: "invitee", Nickname: "invitee_nick"}, nil
		},
	}

	uc := newInviteUseCase(teamRepo, memberRepo, &mockInvitationRepository{}, up)
	out, err := uc.Execute(ctx(), InviteToTeamInput{
		CurrentUser: asContestant("requester"),
		TeamID:      "t1",
		Nickname:    "invitee_nick",
	}, time.Now())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.InviteeUser.ID != "invitee" {
		t.Errorf("InviteeUser.ID = %q, want invitee", out.InviteeUser.ID)
	}
	if out.TeamID != "t1" {
		t.Errorf("TeamID = %q, want t1", out.TeamID)
	}
}
