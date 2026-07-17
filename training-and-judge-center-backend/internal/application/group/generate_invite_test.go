package group

import (
	"context"
	"errors"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const testFrontendBaseURL = "http://localhost:5173"

func newGenerateInviteUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	nicknameResolver NicknameResolver,
	emailResolver EmailResolver,
	userProvider UserProvider,
	emailSender appshared.EmailSender,
) *GenerateInviteUseCase {
	return NewGenerateInviteUseCase(groupRepo, memberRepo, invitationRepo, nicknameResolver, emailResolver, userProvider, &mockTransactionManager{}, emailSender, testFrontendBaseURL)
}

func TestGenerateInvite_EmptyGroupIDReturnsValidationError(t *testing.T) {
	uc := newGenerateInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, &mockEmailSender{})

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
	uc := newGenerateInviteUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, &mockEmailSender{})

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "nonexistent",
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestGenerateInvite_CallerNotLeadReturns403(t *testing.T) {
	g := inviteGroup(t)
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{}, // caller is not a member
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
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

func TestGenerateInvite_OpenPolicyReturns400(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInvalidJoinPolicy {
		t.Fatalf("expected INVALID_JOIN_POLICY, got %v", err)
	}
	if ae.Kind != apperror.KindBadRequest {
		t.Errorf("expected kind BAD_REQUEST, got %s", ae.Kind)
	}
}

// TestGenerateInvite_PolicyChangedInsideTx_ReturnsBadRequest verifies the
// join-policy re-check inside the transaction: a concurrent UpdateGroup that
// switches the group away from INVITE mode between the pre-tx check and the
// transaction's commit must not leave a dangling invitation.
func TestGenerateInvite_PolicyChangedInsideTx_ReturnsBadRequest(t *testing.T) {
	repo := &mockGroupRepository{groups: []*domainGroup.Group{inviteGroup(t)}}
	invRepo := &mockInvitationRepository{}

	txCalled := false
	txManager := &mockTransactionManager{
		withTxFn: func(ctx context.Context, fn func(txCtx context.Context) error) error {
			txCalled = true
			// Simulate a concurrent UpdateGroup committing between the pre-tx
			// policy check and this transaction.
			repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Invite Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
			return fn(ctx)
		},
	}
	uc := NewGenerateInviteUseCase(repo, leadMemberRepo("g1", "u1"), invRepo, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, txManager, &mockEmailSender{}, testFrontendBaseURL)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if !txCalled {
		t.Fatal("expected WithTx to be called")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInvalidJoinPolicy {
		t.Fatalf("expected INVALID_JOIN_POLICY, got %v", err)
	}
	if len(invRepo.savedInvitations) != 0 {
		t.Error("expected no invitation to be saved when the policy changed concurrently")
	}
}

func TestGenerateInvite_MultipleIdentifiersReturnsValidationError(t *testing.T) {
	g := inviteGroup(t)
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:      "g1",
		UserNickname: "bob",
		UserEmail:    "bob@example.com",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestGenerateInvite_NicknameNotFoundReturns404(t *testing.T) {
	g := inviteGroup(t)
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		&mockNicknameResolver{}, // no user
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:      "g1",
		UserNickname: "ghost",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeNicknameNotFound {
		t.Fatalf("expected NICKNAME_NOT_FOUND, got %v", err)
	}
}

func TestGenerateInvite_EmailNotFoundReturns404(t *testing.T) {
	g := inviteGroup(t)
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailResolver{}, // no user
		&mockUserProvider{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		UserEmail:   "ghost@example.com",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeEmailNotFound {
		t.Fatalf("expected EMAIL_NOT_FOUND, got %v", err)
	}
}

func TestGenerateInvite_UserIDNotFoundReturns404(t *testing.T) {
	g := inviteGroup(t)
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{}, // empty displays map
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		UserID:      "ghost-id",
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeNicknameNotFound {
		t.Fatalf("expected NICKNAME_NOT_FOUND (generic user-not-found), got %v", err)
	}
}

func TestGenerateInvite_GeneralInvitationSuccess(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitation.InviteeID != nil {
		t.Errorf("expected general invitation (nil InviteeID), got %v", *out.Invitation.InviteeID)
	}
	if out.Invitee != nil {
		t.Errorf("expected nil Invitee for general invitation, got %+v", out.Invitee)
	}
	if len(invRepo.savedInvitations) != 1 {
		t.Fatalf("expected 1 saved invitation, got %d", len(invRepo.savedInvitations))
	}
}

func TestGenerateInvite_PersonalInvitationByNicknameSuccess(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	display := &UserDisplay{ID: "invitee-1", Nickname: "bob"}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{user: display},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:      "g1",
		UserNickname: "bob",
		CurrentUser:  asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitation.InviteeID == nil || *out.Invitation.InviteeID != "invitee-1" {
		t.Errorf("expected InviteeID = invitee-1, got %v", out.Invitation.InviteeID)
	}
	if out.Invitee != display {
		t.Errorf("expected Invitee = %+v, got %+v", display, out.Invitee)
	}
}

func TestGenerateInvite_PersonalInvitationByEmailSuccess(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	display := &UserDisplay{ID: "invitee-2", Nickname: "carol"}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{user: display},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		UserEmail:   "carol@example.com",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitation.InviteeID == nil || *out.Invitation.InviteeID != "invitee-2" {
		t.Errorf("expected InviteeID = invitee-2, got %v", out.Invitation.InviteeID)
	}
}

func TestGenerateInvite_PersonalInvitationByUserIDSuccess(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	display := &UserDisplay{ID: "invitee-3", Nickname: "dan"}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{displays: map[string]*UserDisplay{"invitee-3": display}},
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		UserID:      "invitee-3",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitation.InviteeID == nil || *out.Invitation.InviteeID != "invitee-3" {
		t.Errorf("expected InviteeID = invitee-3, got %v", out.Invitation.InviteeID)
	}
}

func TestGenerateInvite_ExistingPendingInvitationIsRevokedThenNewCreated(t *testing.T) {
	g := inviteGroup(t)
	inviteeID := mustUID("invitee-1")
	existing := mustInvitation(t, "old-inv", "g1", &inviteeID, "u1")
	invRepo := &mockInvitationRepository{
		byID:          map[string]*domainGroup.GroupInvitation{"old-inv": existing},
		pendingResult: existing,
	}
	display := &UserDisplay{ID: "invitee-1", Nickname: "bob"}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{user: display},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:      "g1",
		UserNickname: "bob",
		CurrentUser:  asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invRepo.transitions) != 1 {
		t.Fatalf("expected 1 transition (revoke old), got %d", len(invRepo.transitions))
	}
	tr := invRepo.transitions[0]
	if tr.id != "old-inv" || tr.from != domainGroup.InvitationStatusPending || tr.to != domainGroup.InvitationStatusRevoked {
		t.Errorf("expected transition old-inv PENDING->REVOKED, got %+v", tr)
	}
	if len(invRepo.savedInvitations) != 1 || invRepo.savedInvitations[0].ID() == "old-inv" {
		t.Errorf("expected a new invitation to be saved, got %+v", invRepo.savedInvitations)
	}
	if out.Invitation.ID == "old-inv" {
		t.Error("expected output invitation to be the new one, not the revoked one")
	}
}

func TestGenerateInvite_SaveFailurePropagatesError(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{saveErr: errors.New("db failure")}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err == nil {
		t.Fatal("expected error from Save, got nil")
	}
}

func TestGenerateInvite_AdminBypassesMemberCheck(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{}, // admin is not even a member
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asAdmin("admin1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(invRepo.savedInvitations) != 1 {
		t.Fatalf("expected 1 saved invitation, got %d", len(invRepo.savedInvitations))
	}
}

func TestGenerateInvite_ShapeInviteeIDByUserIDMatchesInput(t *testing.T) {
	// The userId branch trusts the provided identifier for InviteeID even
	// when the UserProvider's display map is keyed differently; verifies
	// shared.RestoreUserID(input.UserID) is used, not the display's own ID field.
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	display := &UserDisplay{ID: "invitee-4", Nickname: "eve"}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{displays: map[string]*UserDisplay{"invitee-4": display}},
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		UserID:      "invitee-4",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitation.InviteeID == nil || *out.Invitation.InviteeID != "invitee-4" {
		t.Errorf("expected InviteeID = invitee-4, got %v", out.Invitation.InviteeID)
	}
}

func TestGenerateInvite_TargetedInvite_SendsEmail(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	display := &UserDisplay{ID: "invitee-1", Nickname: "bob", Name: "Bob", Email: "bob@example.com"}
	emailSender := &mockEmailSender{}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{user: display},
		&mockEmailResolver{},
		&mockUserProvider{},
		emailSender,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:      "g1",
		UserNickname: "bob",
		CurrentUser:  asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emailSender.sentMsgs) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(emailSender.sentMsgs))
	}
	if emailSender.sentMsgs[0].To != display.Email {
		t.Errorf("expected email To = %s, got %s", display.Email, emailSender.sentMsgs[0].To)
	}
}

func TestGenerateInvite_GeneralInvite_DoesNotSendEmail(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	emailSender := &mockEmailSender{}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{},
		&mockEmailResolver{},
		&mockUserProvider{},
		emailSender,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emailSender.sentMsgs) != 0 {
		t.Errorf("expected no email sent for a general invitation, got %d", len(emailSender.sentMsgs))
	}
}

func TestGenerateInvite_EmailSendFails_ReturnsServiceUnavailable(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	display := &UserDisplay{ID: "invitee-1", Nickname: "bob", Name: "Bob", Email: "bob@example.com"}
	emailSender := &mockEmailSender{
		sendFn: func(_ context.Context, _ appshared.EmailMessage) error {
			return errors.New("smtp down")
		},
	}
	uc := newGenerateInviteUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		&mockNicknameResolver{user: display},
		&mockEmailResolver{},
		&mockUserProvider{},
		emailSender,
	)

	_, err := uc.Execute(context.Background(), GenerateInviteInput{
		GroupID:      "g1",
		UserNickname: "bob",
		CurrentUser:  asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeEmailDeliveryFailed {
		t.Fatalf("expected EMAIL_DELIVERY_FAILED, got %v", err)
	}
	if ae.Kind != apperror.KindServiceUnavailable {
		t.Errorf("expected kind SERVICE_UNAVAILABLE, got %s", ae.Kind)
	}
	// Commit happens before the email attempt (see generate_invite.go) — the
	// invitation should remain persisted despite the send failure.
	if len(invRepo.savedInvitations) != 1 {
		t.Errorf("expected the invitation to remain persisted despite the email failure, got %d saved", len(invRepo.savedInvitations))
	}
}
