package group_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func testInvitee() *shared.UserID {
	id := shared.RestoreUserID("invitee-1")
	return &id
}

func validInvitation(t *testing.T) *group.GroupInvitation {
	t.Helper()
	inv, err := group.NewGroupInvitation("inv-1", "g-1", testInvitee(), shared.RestoreUserID("lead-1"), testNow)
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	return inv
}

func validGeneralInvitation(t *testing.T) *group.GroupInvitation {
	t.Helper()
	inv, err := group.NewGroupInvitation("inv-1", "g-1", nil, shared.RestoreUserID("lead-1"), testNow)
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	return inv
}

func TestNewGroupInvitation_EmptyID(t *testing.T) {
	_, err := group.NewGroupInvitation("", "g-1", testInvitee(), shared.RestoreUserID("lead-1"), testNow)
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestNewGroupInvitation_EmptyGroupID(t *testing.T) {
	_, err := group.NewGroupInvitation("inv-1", "", testInvitee(), shared.RestoreUserID("lead-1"), testNow)
	if err == nil {
		t.Fatal("expected error for empty groupID")
	}
}

func TestNewGroupInvitation_EmptyInvitedBy(t *testing.T) {
	_, err := group.NewGroupInvitation("inv-1", "g-1", testInvitee(), shared.RestoreUserID(""), testNow)
	if err == nil {
		t.Fatal("expected error for empty invitedBy")
	}
}

func TestNewGroupInvitation_NilInviteeAllowed(t *testing.T) {
	inv := validGeneralInvitation(t)
	if inv.HasInvitee() {
		t.Error("expected general invitation to have no invitee")
	}
	if inv.InviteeID() != nil {
		t.Error("expected InviteeID() to be nil for a general invitation")
	}
}

func TestNewGroupInvitation_StartsAsPending(t *testing.T) {
	inv := validInvitation(t)
	if inv.Status() != group.InvitationStatusPending {
		t.Errorf("expected PENDING, got %s", inv.Status())
	}
}

func TestNewGroupInvitation_ExpiresAtIs72HoursOut(t *testing.T) {
	inv := validInvitation(t)
	want := testNow.UTC().Add(group.InvitationExpiryDuration)
	if !inv.ExpiresAt().Equal(want) {
		t.Errorf("ExpiresAt() = %v, want %v", inv.ExpiresAt(), want)
	}
}

func TestNewGroupInvitation_HasInviteeWhenSet(t *testing.T) {
	inv := validInvitation(t)
	if !inv.HasInvitee() {
		t.Error("expected HasInvitee() to be true when an invitee was provided")
	}
	if inv.InviteeID() == nil || inv.InviteeID().Value() != "invitee-1" {
		t.Errorf("unexpected InviteeID(): %v", inv.InviteeID())
	}
}

func TestAccept_PendingBecomesAccepted(t *testing.T) {
	inv := validInvitation(t)
	if err := inv.Accept(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status() != group.InvitationStatusAccepted {
		t.Errorf("expected ACCEPTED, got %s", inv.Status())
	}
}

func TestAccept_AlreadyAcceptedReturnsError(t *testing.T) {
	inv := validInvitation(t)
	_ = inv.Accept()

	err := inv.Accept()
	if err == nil {
		t.Fatal("expected error when accepting an already-accepted invitation")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeInvitationAlreadyProcessed {
		t.Errorf("expected INVITATION_ALREADY_PROCESSED, got %v", err)
	}
}

func TestRevoke_PendingBecomesRevoked(t *testing.T) {
	inv := validInvitation(t)
	if err := inv.Revoke(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status() != group.InvitationStatusRevoked {
		t.Errorf("expected REVOKED, got %s", inv.Status())
	}
}

func TestRevoke_AlreadyAcceptedReturnsError(t *testing.T) {
	inv := validInvitation(t)
	_ = inv.Accept()

	err := inv.Revoke()
	if err == nil {
		t.Fatal("expected error when revoking an already-accepted invitation")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeInvitationAlreadyProcessed {
		t.Errorf("expected INVITATION_ALREADY_PROCESSED, got %v", err)
	}
}

func TestExpire_PendingBecomesExpired(t *testing.T) {
	inv := validInvitation(t)
	if err := inv.Expire(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.Status() != group.InvitationStatusExpired {
		t.Errorf("expected EXPIRED, got %s", inv.Status())
	}
}

func TestExpire_AlreadyRevokedReturnsError(t *testing.T) {
	inv := validInvitation(t)
	_ = inv.Revoke()

	err := inv.Expire()
	if err == nil {
		t.Fatal("expected error when expiring an already-revoked invitation")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeInvitationAlreadyProcessed {
		t.Errorf("expected INVITATION_ALREADY_PROCESSED, got %v", err)
	}
}

func TestIsExpired(t *testing.T) {
	inv := validInvitation(t) // expiresAt = testNow + 72h

	beforeExpiry := testNow.Add(71 * time.Hour)
	if inv.IsExpired(beforeExpiry) {
		t.Error("expected not expired 1h before the 72h TTL")
	}

	afterExpiry := testNow.Add(73 * time.Hour)
	if !inv.IsExpired(afterExpiry) {
		t.Error("expected expired 1h after the 72h TTL")
	}
}

func TestRestoreGroupInvitation_PreservesFields(t *testing.T) {
	invitee := testInvitee()
	inv := group.RestoreGroupInvitation(
		"inv-1", "g-1", invitee, shared.RestoreUserID("lead-1"),
		group.InvitationStatusAccepted, testNow.Add(72*time.Hour), testNow,
	)
	if inv.ID() != "inv-1" || inv.GroupID() != "g-1" {
		t.Errorf("unexpected identity fields: %+v", inv)
	}
	if inv.Status() != group.InvitationStatusAccepted {
		t.Errorf("expected ACCEPTED, got %s", inv.Status())
	}
	if inv.InviteeID() == nil || inv.InviteeID().Value() != "invitee-1" {
		t.Errorf("unexpected InviteeID(): %v", inv.InviteeID())
	}
}
