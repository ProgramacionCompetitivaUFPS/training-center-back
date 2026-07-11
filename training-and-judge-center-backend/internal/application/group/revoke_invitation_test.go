package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRevokeInvitationUseCase(memberRepo *mockMemberRepository, invRepo *mockInvitationRepository) *RevokeInvitationUseCase {
	return NewRevokeInvitationUseCase(memberRepo, invRepo)
}

func TestRevokeInvitation_NonLeadReturns403(t *testing.T) {
	uc := newRevokeInvitationUseCase(&mockMemberRepository{}, &mockInvitationRepository{})

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("nobody"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestRevokeInvitation_NotFoundReturns404(t *testing.T) {
	uc := newRevokeInvitationUseCase(leadMemberRepo("g1", "lead-id"), &mockInvitationRepository{})

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "nonexistent",
		CurrentUser:  asContestant("lead-id"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationNotFound {
		t.Fatalf("expected INVITATION_NOT_FOUND, got %v", err)
	}
}

func TestRevokeInvitation_CrossGroupInvitationReturns404(t *testing.T) {
	inv := mustInvitation(t, "inv1", "other-group", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newRevokeInvitationUseCase(leadMemberRepo("g1", "lead-id"), invRepo)

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("lead-id"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationNotFound {
		t.Fatalf("expected INVITATION_NOT_FOUND for cross-group invitation, got %v", err)
	}
	if len(invRepo.transitions) != 0 {
		t.Errorf("expected no transition for a cross-group invitation, got %+v", invRepo.transitions)
	}
}

func TestRevokeInvitation_AlreadyProcessedReturns400(t *testing.T) {
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	if err := inv.Revoke(); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newRevokeInvitationUseCase(leadMemberRepo("g1", "lead-id"), invRepo)

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("lead-id"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationAlreadyProcessed {
		t.Fatalf("expected INVITATION_ALREADY_PROCESSED, got %v", err)
	}
}

func TestRevokeInvitation_Success(t *testing.T) {
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newRevokeInvitationUseCase(leadMemberRepo("g1", "lead-id"), invRepo)

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invRepo.transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(invRepo.transitions))
	}
	tr := invRepo.transitions[0]
	if tr.id != "inv1" || tr.from != domainGroup.InvitationStatusPending || tr.to != domainGroup.InvitationStatusRevoked {
		t.Errorf("expected transition inv1 PENDING->REVOKED, got %+v", tr)
	}
}

func TestRevokeInvitation_AdminBypassesMemberCheck(t *testing.T) {
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newRevokeInvitationUseCase(&mockMemberRepository{}, invRepo) // admin is not even a member

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asAdmin("admin1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRevokeInvitation_TransitionFailurePropagatesError(t *testing.T) {
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{
		byID:          map[string]*domainGroup.GroupInvitation{"inv1": inv},
		transitionErr: apperror.NewConflict(domainGroup.ErrCodeInvitationAlreadyProcessed, "already processed"),
	}
	uc := newRevokeInvitationUseCase(leadMemberRepo("g1", "lead-id"), invRepo)

	err := uc.Execute(context.Background(), RevokeInvitationInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("lead-id"),
	})

	if err == nil {
		t.Fatal("expected error from TransitionStatus, got nil")
	}
}
