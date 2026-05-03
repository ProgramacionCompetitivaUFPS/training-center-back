package group

import (
	"context"
	"errors"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestAcceptInvite_EmptyTokenReturnsValidationError(t *testing.T) {
	uc := NewAcceptInviteUseCase(&fakeRepo{}, &fakeMemberRepo{}, &fakeInvitationSvc{})

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "token" {
		t.Errorf("expected field error on 'token', got %+v", ae.Details)
	}
}

func TestAcceptInvite_InvalidTokenPropagatesError(t *testing.T) {
	invalidErr := apperror.NewBadRequest(domainGroup.ErrCodeInvalidInviteToken, "invalid invitation token")
	svc := &fakeInvitationSvc{validateErr: invalidErr}
	uc := NewAcceptInviteUseCase(&fakeRepo{}, &fakeMemberRepo{}, svc)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "bad.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvalidInviteToken {
		t.Fatalf("expected INVALID_INVITE_TOKEN, got %v", err)
	}
}

func TestAcceptInvite_ExpiredTokenPropagatesError(t *testing.T) {
	expiredErr := apperror.NewBadRequest(domainGroup.ErrCodeExpiredInviteToken, "invitation link has expired")
	svc := &fakeInvitationSvc{validateErr: expiredErr}
	uc := NewAcceptInviteUseCase(&fakeRepo{}, &fakeMemberRepo{}, svc)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "expired.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeExpiredInviteToken {
		t.Fatalf("expected EXPIRED_INVITE_TOKEN, got %v", err)
	}
}

func TestAcceptInvite_GroupNotFoundReturns404(t *testing.T) {
	svc := &fakeInvitationSvc{claims: &InvitationClaims{GroupID: "nonexistent"}}
	uc := NewAcceptInviteUseCase(&fakeRepo{}, &fakeMemberRepo{}, svc)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "valid.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestAcceptInvite_PolicyChangedReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	svc := &fakeInvitationSvc{claims: &InvitationClaims{GroupID: "g1"}}
	uc := NewAcceptInviteUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, &fakeMemberRepo{}, svc)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "valid.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.StatusCode != 403 {
		t.Errorf("expected HTTP 403, got %d", ae.StatusCode)
	}
}

func TestAcceptInvite_AlreadyMemberReturns409(t *testing.T) {
	g := inviteGroup(t)
	userID := shared.RestoreUserID("u1")
	existing := domainGroup.RestoreGroupMember("m1", "g1", userID, domainGroup.MemberRoleMember, time.Now())
	memberRepo := &fakeMemberRepo{
		memberships: map[string]*domainGroup.GroupMember{
			keyOf("g1", userID): existing,
		},
	}
	svc := &fakeInvitationSvc{claims: &InvitationClaims{GroupID: "g1"}}
	uc := NewAcceptInviteUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, memberRepo, svc)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "valid.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
	if ae.StatusCode != 409 {
		t.Errorf("expected HTTP 409, got %d", ae.StatusCode)
	}
}

func TestAcceptInvite_SuccessCreatesMemberWithRoleMember(t *testing.T) {
	g := inviteGroup(t)
	memberRepo := &fakeMemberRepo{}
	svc := &fakeInvitationSvc{claims: &InvitationClaims{GroupID: "g1"}}
	uc := NewAcceptInviteUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, memberRepo, svc)

	out, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "valid.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Member == nil {
		t.Fatal("expected non-nil member")
	}
	if out.Member.Role() != domainGroup.MemberRoleMember {
		t.Errorf("Role = %v, want MEMBER", out.Member.Role())
	}
	if out.Member.GroupID() != "g1" {
		t.Errorf("GroupID = %q, want %q", out.Member.GroupID(), "g1")
	}
	if out.Member.UserID().Value() != "u1" {
		t.Errorf("UserID = %q, want %q", out.Member.UserID().Value(), "u1")
	}
	if memberRepo.savedMember == nil {
		t.Error("expected Save to be called with the new member")
	}
}

func TestAcceptInvite_SaveFailurePropagatesError(t *testing.T) {
	g := inviteGroup(t)
	memberRepo := &fakeMemberRepo{saveErr: errors.New("db failure")}
	svc := &fakeInvitationSvc{claims: &InvitationClaims{GroupID: "g1"}}
	uc := NewAcceptInviteUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, memberRepo, svc)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		Token:       "valid.token",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	if err == nil {
		t.Fatal("expected error from Save, got nil")
	}
}
