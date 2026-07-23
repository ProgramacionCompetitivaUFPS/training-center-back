package group

import (
	"context"
	"errors"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newInviteByNicknamesUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	nicknameResolver NicknameResolver,
	emailSender *mockEmailSender,
) *InviteByNicknamesUseCase {
	return NewInviteByNicknamesUseCase(groupRepo, memberRepo, invitationRepo, nicknameResolver, &mockTransactionManager{}, emailSender, testFrontendBaseURL)
}

func TestInviteByNicknames_EmptyGroupIDReturnsValidationError(t *testing.T) {
	// Arrange
	uc := newInviteByNicknamesUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, &mockNicknameResolver{}, &mockEmailSender{})

	// Act
	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "",
		Nicknames:   []string{"bob"},
		CurrentUser: asCoach("u1"),
	})

	// Assert
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestInviteByNicknames_EmptyNicknamesListReturnsValidationError(t *testing.T) {
	uc := newInviteByNicknamesUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, &mockNicknameResolver{}, &mockEmailSender{})

	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{},
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestInviteByNicknames_ExceedsMaxBatchSizeReturnsValidationError(t *testing.T) {
	nicknames := make([]string, MaxInviteBatchSize+1)
	for i := range nicknames {
		nicknames[i] = "user"
	}
	uc := newInviteByNicknamesUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, &mockNicknameResolver{}, &mockEmailSender{})

	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   nicknames,
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestInviteByNicknames_BlankNicknameEntryReturnsValidationError(t *testing.T) {
	uc := newInviteByNicknamesUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockInvitationRepository{}, &mockNicknameResolver{}, &mockEmailSender{})

	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice", "  "},
		CurrentUser: asCoach("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestInviteByNicknames_CallerNotLeadReturns403(t *testing.T) {
	g := inviteGroup(t)
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{}, // caller is not a member
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"bob"},
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestInviteByNicknames_OpenPolicyReturns400(t *testing.T) {
	g := mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		&mockNicknameResolver{},
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"bob"},
		CurrentUser: asContestant("u1"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInvalidJoinPolicy {
		t.Fatalf("expected INVALID_JOIN_POLICY, got %v", err)
	}
}

func TestInviteByNicknames_AllValidNicknames_AllInvitedAndEmailed(t *testing.T) {
	// Arrange
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	emailSender := &mockEmailSender{}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "u-alice", Nickname: "alice", Name: "Alice", Email: "alice@example.com"},
		"bob":   {ID: "u-bob", Nickname: "bob", Name: "Bob", Email: "bob@example.com"},
		"carol": {ID: "u-carol", Nickname: "carol", Name: "Carol", Email: "carol@example.com"},
	}}
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		resolver,
		emailSender,
	)

	// Act
	out, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice", "bob", "carol"},
		CurrentUser: asContestant("u1"),
	})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(out.Results))
	}
	for _, r := range out.Results {
		if r.Status != InviteResultInvited {
			t.Errorf("expected status invited for %s, got %s (%s)", r.Nickname, r.Status, r.Reason)
		}
	}
	if len(emailSender.sentMsgs) != 3 {
		t.Errorf("expected 3 emails sent, got %d", len(emailSender.sentMsgs))
	}
	if len(invRepo.savedInvitations) != 3 {
		t.Errorf("expected 3 invitations persisted, got %d", len(invRepo.savedInvitations))
	}
}

func TestInviteByNicknames_UnknownNickname_MarkedFailedWithoutAbortingBatch(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "u-alice", Nickname: "alice", Name: "Alice", Email: "alice@example.com"},
		"carol": {ID: "u-carol", Nickname: "carol", Name: "Carol", Email: "carol@example.com"},
	}}
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		resolver,
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice", "ghost", "carol"},
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(out.Results))
	}
	byNickname := make(map[string]InviteByNicknamesResult, len(out.Results))
	for _, r := range out.Results {
		byNickname[r.Nickname] = r
	}
	if byNickname["alice"].Status != InviteResultInvited || byNickname["carol"].Status != InviteResultInvited {
		t.Errorf("expected alice and carol to be invited, got %+v", out.Results)
	}
	if byNickname["ghost"].Status != InviteResultFailed || byNickname["ghost"].Reason != "no user found with that nickname" {
		t.Errorf("expected ghost to be failed with a not-found reason, got %+v", byNickname["ghost"])
	}
	if len(invRepo.savedInvitations) != 2 {
		t.Errorf("expected 2 invitations persisted, got %d", len(invRepo.savedInvitations))
	}
}

func TestInviteByNicknames_ResolverError_MarkedFailedWithDistinctReason(t *testing.T) {
	// A resolver error (e.g. a DB timeout) is NOT the same thing as a
	// genuinely unknown nickname — it must not be reported with the same
	// "no user found" reason, or the caller can't tell a typo from a
	// transient infrastructure failure worth retrying.
	g := inviteGroup(t)
	resolver := &mockNicknameResolver{err: errors.New("db timeout")}
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		&mockInvitationRepository{},
		resolver,
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice"},
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	r := out.Results[0]
	if r.Status != InviteResultFailed {
		t.Errorf("expected status failed, got %s", r.Status)
	}
	if r.Reason != "nickname lookup failed" {
		t.Errorf("expected reason 'nickname lookup failed', got %q", r.Reason)
	}
}

func TestInviteByNicknames_DuplicateNicknamesCaseInsensitive_Deduplicated(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"Alice": {ID: "u-alice", Nickname: "Alice", Name: "Alice", Email: "alice@example.com"},
	}}
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		resolver,
		&mockEmailSender{},
	)

	out, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"Alice", "alice"},
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected duplicates to collapse into 1 result, got %d", len(out.Results))
	}
}

func TestInviteByNicknames_TxFailureForOneNickname_OthersStillSucceed(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "u-alice", Nickname: "alice", Name: "Alice", Email: "alice@example.com"},
		"bob":   {ID: "u-bob", Nickname: "bob", Name: "Bob", Email: "bob@example.com"},
		"carol": {ID: "u-carol", Nickname: "carol", Name: "Carol", Email: "carol@example.com"},
	}}

	call := 0
	txManager := &mockTransactionManager{
		withTxFn: func(ctx context.Context, fn func(txCtx context.Context) error) error {
			call++
			if call == 2 {
				return errors.New("db failure")
			}
			return fn(ctx)
		},
	}
	uc := NewInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		resolver,
		txManager,
		&mockEmailSender{},
		testFrontendBaseURL,
	)

	out, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice", "bob", "carol"},
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	byNickname := make(map[string]InviteByNicknamesResult, len(out.Results))
	for _, r := range out.Results {
		byNickname[r.Nickname] = r
	}
	if byNickname["alice"].Status != InviteResultInvited || byNickname["carol"].Status != InviteResultInvited {
		t.Errorf("expected alice and carol to be invited despite bob's tx failure, got %+v", out.Results)
	}
	if byNickname["bob"].Status != InviteResultFailed {
		t.Errorf("expected bob to be failed, got %s", byNickname["bob"].Status)
	}
	if len(invRepo.savedInvitations) != 2 {
		t.Errorf("expected 2 invitations persisted (alice, carol), got %d", len(invRepo.savedInvitations))
	}
}

func TestInviteByNicknames_EmailSendFails_MarkedEmailFailedButInvitationPersists(t *testing.T) {
	g := inviteGroup(t)
	invRepo := &mockInvitationRepository{}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "u-alice", Nickname: "alice", Name: "Alice", Email: "alice@example.com"},
	}}
	emailSender := &mockEmailSender{
		sendFn: func(_ context.Context, _ appshared.EmailMessage) error {
			return errors.New("smtp down")
		},
	}
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		resolver,
		emailSender,
	)

	out, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice"},
		CurrentUser: asContestant("u1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out.Results))
	}
	r := out.Results[0]
	if r.Status != InviteResultEmailFailed {
		t.Errorf("expected status email_failed, got %s", r.Status)
	}
	if r.Invitation == nil {
		t.Error("expected Invitation to be populated even though the email failed")
	}
	if len(invRepo.savedInvitations) != 1 {
		t.Errorf("expected the invitation to remain persisted despite the email failure, got %d saved", len(invRepo.savedInvitations))
	}
}

func TestInviteByNicknames_ExistingPendingInvitation_IsRevokedAndReplaced(t *testing.T) {
	g := inviteGroup(t)
	inviteeID := mustUID("u-alice")
	existing := mustInvitation(t, "old-inv", "g1", &inviteeID, "u1")
	invRepo := &mockInvitationRepository{
		byID:          map[string]*domainGroup.GroupInvitation{"old-inv": existing},
		pendingResult: existing,
	}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "u-alice", Nickname: "alice", Name: "Alice", Email: "alice@example.com"},
	}}
	uc := newInviteByNicknamesUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "u1"),
		invRepo,
		resolver,
		&mockEmailSender{},
	)

	_, err := uc.Execute(context.Background(), InviteByNicknamesInput{
		GroupID:     "g1",
		Nicknames:   []string{"alice"},
		CurrentUser: asContestant("u1"),
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
}
