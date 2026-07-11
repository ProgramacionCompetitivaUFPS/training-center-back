package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newAcceptInviteUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	txManager *mockTransactionManager,
) *AcceptInviteUseCase {
	if txManager == nil {
		txManager = &mockTransactionManager{}
	}
	return NewAcceptInviteUseCase(groupRepo, memberRepo, invitationRepo, txManager)
}

func TestAcceptInvite_EmptyInvitationIDReturnsValidationError(t *testing.T) {
	uc := newAcceptInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		InvitationID: "",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "invitationId" {
		t.Errorf("expected field error on 'invitationId', got %+v", ae.Details)
	}
}

func TestAcceptInvite_InvitationNotFoundReturns404(t *testing.T) {
	uc := newAcceptInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		InvitationID: "nonexistent",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationNotFound {
		t.Fatalf("expected INVITATION_NOT_FOUND, got %v", err)
	}
}

func TestAcceptInvite_CrossGroupInvitationReturns404(t *testing.T) {
	// The invitation belongs to group "g1", but the request targets group "g2"
	// (e.g. the URL's {groupId} doesn't match the invitation's actual group).
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newAcceptInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g2",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationNotFound {
		t.Fatalf("expected INVITATION_NOT_FOUND, got %v", err)
	}
}

func TestAcceptInvite_WrongInviteeReturns403(t *testing.T) {
	g := inviteGroup(t)
	inviteeID := mustUID("invitee-1")
	inv := mustInvitation(t, "inv1", "g1", &inviteeID, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{}, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("someone-else"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", ae.Kind)
	}
}

func TestAcceptInvite_AlreadyProcessedReturns400(t *testing.T) {
	g := inviteGroup(t)
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	if err := inv.Revoke(); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{}, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationAlreadyProcessed {
		t.Fatalf("expected INVITATION_ALREADY_PROCESSED, got %v", err)
	}
	if ae.Kind != apperror.KindBadRequest {
		t.Errorf("expected kind BAD_REQUEST, got %s", ae.Kind)
	}
}

func TestAcceptInvite_ExpiredInvitationReturns400AndFlipsStatus(t *testing.T) {
	g := inviteGroup(t)
	longAgo := testNow
	inv, err := domainGroup.NewGroupInvitation("inv1", "g1", nil, shared.RestoreUserID("lead1"), longAgo)
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{}, invRepo, nil)

	// well past the 72h expiry window
	_, err = uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationExpired {
		t.Fatalf("expected INVITATION_EXPIRED, got %v", err)
	}
	if len(invRepo.transitions) != 1 {
		t.Fatalf("expected 1 transition (expire), got %d", len(invRepo.transitions))
	}
	tr := invRepo.transitions[0]
	if tr.id != "inv1" || tr.from != domainGroup.InvitationStatusPending || tr.to != domainGroup.InvitationStatusExpired {
		t.Errorf("expected transition inv1 PENDING->EXPIRED, got %+v", tr)
	}
}

func TestAcceptInvite_PolicyChangedReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{}, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
	if ae.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", ae.Kind)
	}
}

func TestAcceptInvite_AlreadyMemberReturns409(t *testing.T) {
	g := inviteGroup(t)
	userID := shared.RestoreUserID("u1")
	existing := domainGroup.RestoreGroupMember("m1", "g1", userID, domainGroup.MemberRoleMember, testNow, nil, domainGroup.JoinMethodOpenJoin)
	memberRepo := &mockMemberRepository{
		memberships: map[string]*domainGroup.GroupMember{
			keyOf("g1", userID): existing,
		},
	}
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
	if ae.Kind != apperror.KindConflict {
		t.Errorf("expected kind CONFLICT, got %s", ae.Kind)
	}
}

func TestAcceptInvite_GeneralInvitationSuccessAndStaysPending(t *testing.T) {
	g := inviteGroup(t)
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	memberRepo := &mockMemberRepository{}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, invRepo, nil)

	out, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Member.Role != domainGroup.MemberRoleMember.String() {
		t.Errorf("Role = %v, want MEMBER", out.Member.Role)
	}
	if memberRepo.savedMember == nil {
		t.Fatal("expected Save to be called with the new member")
	}
	if len(invRepo.transitions) != 0 {
		t.Errorf("expected general invitation to stay PENDING (no transition), got %+v", invRepo.transitions)
	}
}

func TestAcceptInvite_PersonalInvitationSuccessTransitionsToAccepted(t *testing.T) {
	g := inviteGroup(t)
	inviteeID := mustUID("u1")
	inv := mustInvitation(t, "inv1", "g1", &inviteeID, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	memberRepo := &mockMemberRepository{}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invRepo.transitions) != 1 {
		t.Fatalf("expected 1 transition (accept), got %d", len(invRepo.transitions))
	}
	tr := invRepo.transitions[0]
	if tr.id != "inv1" || tr.from != domainGroup.InvitationStatusPending || tr.to != domainGroup.InvitationStatusAccepted {
		t.Errorf("expected transition inv1 PENDING->ACCEPTED, got %+v", tr)
	}
}

func TestAcceptInvite_SaveFailurePropagatesError(t *testing.T) {
	g := inviteGroup(t)
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	memberRepo := &mockMemberRepository{saveErr: errors.New("db failure")}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, invRepo, nil)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	if err == nil {
		t.Fatal("expected error from Save, got nil")
	}
}

// TestAcceptInvite_RaceCondition_MembershipCheckedInsideTx verifies the fix
// for the TOCTOU race described in the DoD: the membership-existence check
// happens inside the txManager.WithTx callback, not before it. This test
// simulates another concurrent request winning the race by inserting the
// membership row at the moment WithTx starts its callback — a check done
// before the transaction (the old buggy sequence) would miss this and still
// call Save, producing a duplicate membership.
func TestAcceptInvite_RaceCondition_MembershipCheckedInsideTx(t *testing.T) {
	g := inviteGroup(t)
	inviteeID := mustUID("u1")
	inv := mustInvitation(t, "inv1", "g1", &inviteeID, "lead1")
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	memberRepo := &mockMemberRepository{}

	txCalled := false
	txManager := &mockTransactionManager{
		withTxFn: func(ctx context.Context, fn func(txCtx context.Context) error) error {
			txCalled = true
			winningMember := domainGroup.RestoreGroupMember(
				"m-other", "g1", mustUID("u1"), domainGroup.MemberRoleMember, testNow, nil, domainGroup.JoinMethodInvitation,
			)
			memberRepo.memberships = map[string]*domainGroup.GroupMember{
				keyOf("g1", mustUID("u1")): winningMember,
			}
			return fn(ctx)
		},
	}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, invRepo, txManager)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	if !txCalled {
		t.Fatal("expected WithTx to be called")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
	if ae.Kind != apperror.KindConflict {
		t.Errorf("expected kind CONFLICT, got %s", ae.Kind)
	}
	if memberRepo.savedMember != nil {
		t.Error("expected Save NOT to be called when the in-tx membership check finds an existing member")
	}
	if len(invRepo.transitions) != 0 {
		t.Error("expected no invitation status transition when membership creation is aborted")
	}
}

// TestAcceptInvite_RaceCondition_GeneralInvitationRevokedBeforeTxCommits
// verifies the fix for a gap in the original race-condition fix: a general
// (invitee-less) invitation never transitions status on accept, so it has no
// compare-and-swap write to catch a concurrent revoke. The invitation must
// instead be re-read and re-checked inside the transaction.
func TestAcceptInvite_RaceCondition_GeneralInvitationRevokedBeforeTxCommits(t *testing.T) {
	g := inviteGroup(t)
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1") // general invitation
	invRepo := &mockInvitationRepository{byID: map[string]*domainGroup.GroupInvitation{"inv1": inv}}
	memberRepo := &mockMemberRepository{}

	txCalled := false
	txManager := &mockTransactionManager{
		withTxFn: func(ctx context.Context, fn func(txCtx context.Context) error) error {
			txCalled = true
			// Simulate a concurrent RevokeInvitationUseCase committing between
			// this request's pre-tx IsPending() check and this transaction.
			if err := inv.Revoke(); err != nil {
				t.Fatalf("Revoke: %v", err)
			}
			return fn(ctx)
		},
	}
	uc := newAcceptInviteUseCase(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, invRepo, txManager)

	_, err := uc.Execute(context.Background(), AcceptInviteInput{
		GroupID:      "g1",
		InvitationID: "inv1",
		CurrentUser:  asContestant("u1"),
	})

	if !txCalled {
		t.Fatal("expected WithTx to be called")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvitationAlreadyProcessed {
		t.Fatalf("expected INVITATION_ALREADY_PROCESSED, got %v", err)
	}
	if ae.Kind != apperror.KindConflict {
		t.Errorf("expected kind CONFLICT, got %s", ae.Kind)
	}
	if memberRepo.savedMember != nil {
		t.Error("expected Save NOT to be called when the invitation was revoked concurrently")
	}
}
