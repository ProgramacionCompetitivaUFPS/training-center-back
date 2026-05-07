package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// --- fakes ---

type fakeInvitationSvc struct {
	token       string
	genErr      error
	claims      *InvitationClaims
	validateErr error
}

func (f *fakeInvitationSvc) GenerateInviteToken(groupID, inviterID string) (string, error) {
	return f.token, f.genErr
}

func (f *fakeInvitationSvc) ValidateInviteToken(token string) (*InvitationClaims, error) {
	return f.claims, f.validateErr
}

// --- helpers ---

func inviteGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Invite Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite)
}

// --- tests ---

func TestGenerateInvite_EmptyGroupIDReturnsValidationError(t *testing.T) {
	uc := NewGenerateInviteUseCase(&fakeRepo{}, &fakeMemberRepo{}, &fakeInvitationSvc{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
	uc := NewGenerateInviteUseCase(&fakeRepo{}, &fakeMemberRepo{}, &fakeInvitationSvc{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "nonexistent",
		CurrentUser: currentUser("u1", shared.RoleCoach),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestGenerateInvite_OpenPolicyReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := NewGenerateInviteUseCase(&fakeRepo{groups: []*domainGroup.Group{g}}, &fakeMemberRepo{}, &fakeInvitationSvc{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleCoach),
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
		&fakeRepo{groups: []*domainGroup.Group{g}},
		&fakeMemberRepo{}, // user is not a member
		&fakeInvitationSvc{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
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
	svc := &fakeInvitationSvc{token: "signed.jwt.token"}
	uc := NewGenerateInviteUseCase(
		&fakeRepo{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		svc,
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
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
	svc := &fakeInvitationSvc{token: "admin.jwt.token"}
	uc := NewGenerateInviteUseCase(
		&fakeRepo{groups: []*domainGroup.Group{g}},
		&fakeMemberRepo{}, // admin bypasses member check
		svc,
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleAdmin),
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
	svc := &fakeInvitationSvc{token: "tok"}
	uc := NewGenerateInviteUseCase(
		&fakeRepo{groups: []*domainGroup.Group{g}},
		&fakeMemberRepo{}, // admin bypasses member check
		svc,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleAdmin),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS for wrong policy, got %v", err)
	}
	if ae.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", ae.Kind)
	}
}

func TestGenerateInvite_ServiceErrorPropagates(t *testing.T) {
	g := inviteGroup(t)
	svc := &fakeInvitationSvc{genErr: errors.New("signing failed")}
	uc := NewGenerateInviteUseCase(
		&fakeRepo{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		svc,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: currentUser("u1", shared.RoleContestant),
	})

	if err == nil {
		t.Fatal("expected error from GenerateInviteToken, got nil")
	}
}
