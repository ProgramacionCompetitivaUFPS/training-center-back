package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// --- tests ---

func TestGenerateInvite_EmptyGroupIDReturnsValidationError(t *testing.T) {
	uc := NewGenerateInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationTokenService{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "",
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "groupId" {
		t.Errorf("expected field error on 'groupId', got %+v", ae.Details)
	}
}

func TestGenerateInvite_GroupNotFoundReturns404(t *testing.T) {
	uc := NewGenerateInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationTokenService{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "nonexistent",
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestGenerateInvite_OpenPolicyReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := NewGenerateInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{}, &mockInvitationTokenService{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", ae.Kind)
	}
}

func TestGenerateInvite_CallerNotLeadReturns403(t *testing.T) {
	g := inviteGroup(t)
	uc := NewGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{}, // user is not a member
		&mockInvitationTokenService{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", ae.Kind)
	}
}

func TestGenerateInvite_LeadReturnsToken(t *testing.T) {
	g := inviteGroup(t)
	svc := &mockInvitationTokenService{token: "signed.jwt.token"}
	uc := NewGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		svc,
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Token != "signed.jwt.token" {
		t.Errorf("Token = %q, want %q", out.Token, "signed.jwt.token")
	}
}

func TestGenerateInvite_AdminReturnsToken(t *testing.T) {
	g := inviteGroup(t)
	svc := &mockInvitationTokenService{token: "admin.jwt.token"}
	uc := NewGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{}, // admin bypasses member check
		svc,
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asAdmin("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Token != "admin.jwt.token" {
		t.Errorf("Token = %q, want %q", out.Token, "admin.jwt.token")
	}
}

func TestGenerateInvite_AdminOnOpenGroupReturns403ForPolicy(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	svc := &mockInvitationTokenService{token: "tok"}
	uc := NewGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{}, // admin bypasses member check
		svc,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asAdmin("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInvalidJoinPolicy {
		t.Fatalf("expected INVALID_JOIN_POLICY for wrong policy, got %v", err)
	}
	if ae.Kind != apperror.KindBadRequest {
		t.Errorf("expected kind BAD_REQUEST, got %s", ae.Kind)
	}
}

func TestGenerateInvite_ServiceErrorPropagates(t *testing.T) {
	g := inviteGroup(t)
	svc := &mockInvitationTokenService{genErr: errors.New("signing failed")}
	uc := NewGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		svc,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err == nil {
		t.Fatal("expected error from GenerateInviteToken, got nil")
	}
}
