package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListGroupInvitationsUseCase(memberRepo *mockMemberRepository, invRepo *mockInvitationRepository, userProvider *mockUserProvider) *ListGroupInvitationsUseCase {
	if userProvider == nil {
		userProvider = &mockUserProvider{}
	}
	return NewListGroupInvitationsUseCase(memberRepo, invRepo, userProvider)
}

func TestListGroupInvitations_NonLeadReturns403(t *testing.T) {
	uc := newListGroupInvitationsUseCase(&mockMemberRepository{}, &mockInvitationRepository{}, nil)

	_, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("nobody"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestListGroupInvitations_AdminCanList(t *testing.T) {
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{findByGroupResult: []*domainGroup.GroupInvitation{inv}, findByGroupTotal: 1}
	uc := newListGroupInvitationsUseCase(&mockMemberRepository{}, invRepo, nil)

	out, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asAdmin("admin"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(out.Invitations))
	}
}

func TestListGroupInvitations_DefaultsToStatusPending(t *testing.T) {
	invRepo := &mockInvitationRepository{}
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), invRepo, nil)

	_, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invRepo.lastFilters.Status == nil || *invRepo.lastFilters.Status != domainGroup.InvitationStatusPending {
		t.Fatalf("expected default status filter PENDING, got %+v", invRepo.lastFilters.Status)
	}
}

func TestListGroupInvitations_InvalidStatusReturnsValidationError(t *testing.T) {
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), &mockInvitationRepository{}, nil)

	_, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		Status:      "INVALID",
		CurrentUser: asContestant("lead-id"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestListGroupInvitations_LimitExceeded100Returns400(t *testing.T) {
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), &mockInvitationRepository{}, nil)

	_, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       101,
		CurrentUser: asContestant("lead-id"),
	})

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestListGroupInvitations_ExpiredPendingShowsEffectiveStatusWithoutWriting(t *testing.T) {
	// A PENDING invitation whose 72h TTL already elapsed (built against
	// testNow, which is far in the past relative to real wall time) must be
	// reported as EXPIRED for display purposes only — the DoD asks for lazy
	// expiration, so ListGroupInvitations must never write the transition.
	staleInv, err := domainGroup.NewGroupInvitation("inv1", "g1", nil, mustUID("lead1"), testNow)
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	invRepo := &mockInvitationRepository{findByGroupResult: []*domainGroup.GroupInvitation{staleInv}, findByGroupTotal: 1}
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), invRepo, nil)

	out, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(out.Invitations))
	}
	detail := out.Invitations[0]
	if detail.Invitation.Status != domainGroup.InvitationStatusPending.String() {
		t.Errorf("expected underlying Status to remain PENDING, got %s", detail.Invitation.Status)
	}
	if detail.EffectiveStatus != domainGroup.InvitationStatusExpired.String() {
		t.Errorf("expected EffectiveStatus EXPIRED, got %s", detail.EffectiveStatus)
	}
	if len(invRepo.transitions) != 0 {
		t.Errorf("expected no status transition to be written, got %+v", invRepo.transitions)
	}
}

func TestListGroupInvitations_NonExpiredPendingKeepsEffectiveStatus(t *testing.T) {
	freshInv, err := domainGroup.NewGroupInvitation("inv1", "g1", nil, mustUID("lead1"), time.Now())
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	invRepo := &mockInvitationRepository{findByGroupResult: []*domainGroup.GroupInvitation{freshInv}, findByGroupTotal: 1}
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), invRepo, nil)

	out, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitations[0].EffectiveStatus != domainGroup.InvitationStatusPending.String() {
		t.Errorf("expected EffectiveStatus PENDING, got %s", out.Invitations[0].EffectiveStatus)
	}
}

func TestListGroupInvitations_ResolvesInviteeAndInviterDisplays(t *testing.T) {
	inviteeID := mustUID("invitee-1")
	inv := mustInvitation(t, "inv1", "g1", &inviteeID, "lead1")
	invRepo := &mockInvitationRepository{findByGroupResult: []*domainGroup.GroupInvitation{inv}, findByGroupTotal: 1}
	userProvider := &mockUserProvider{displays: map[string]*UserDisplay{
		"invitee-1": {ID: "invitee-1", Nickname: "bob"},
		"lead1":     {ID: "lead1", Nickname: "leader"},
	}}
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), invRepo, userProvider)

	out, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	detail := out.Invitations[0]
	if detail.Invitee == nil || detail.Invitee.Nickname != "bob" {
		t.Errorf("expected Invitee nickname 'bob', got %+v", detail.Invitee)
	}
	if detail.InvitedBy == nil || detail.InvitedBy.Nickname != "leader" {
		t.Errorf("expected InvitedBy nickname 'leader', got %+v", detail.InvitedBy)
	}
}

func TestListGroupInvitations_GeneralInvitationHasNilInvitee(t *testing.T) {
	inv := mustInvitation(t, "inv1", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{findByGroupResult: []*domainGroup.GroupInvitation{inv}, findByGroupTotal: 1}
	userProvider := &mockUserProvider{displays: map[string]*UserDisplay{"lead1": {ID: "lead1", Nickname: "leader"}}}
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), invRepo, userProvider)

	out, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Invitations[0].Invitee != nil {
		t.Errorf("expected nil Invitee for general invitation, got %+v", out.Invitations[0].Invitee)
	}
}

func TestListGroupInvitations_DedupesUserIDsForDisplayLookup(t *testing.T) {
	// Same inviter ("lead1") across every row must only appear once in the
	// batch GetDisplays call, not once per invitation row.
	inviteeA := mustUID("invitee-a")
	inviteeB := mustUID("invitee-b")
	inv1 := mustInvitation(t, "inv1", "g1", &inviteeA, "lead1")
	inv2 := mustInvitation(t, "inv2", "g1", &inviteeB, "lead1")
	inv3 := mustInvitation(t, "inv3", "g1", nil, "lead1")
	invRepo := &mockInvitationRepository{
		findByGroupResult: []*domainGroup.GroupInvitation{inv1, inv2, inv3},
		findByGroupTotal:  3,
	}
	userProvider := &mockUserProvider{}
	uc := newListGroupInvitationsUseCase(leadMemberRepo("g1", "lead-id"), invRepo, userProvider)

	_, err := uc.Execute(context.Background(), ListGroupInvitationsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seen := make(map[string]int, len(userProvider.lastCallIDs))
	for _, id := range userProvider.lastCallIDs {
		seen[id]++
	}
	for id, count := range seen {
		if count > 1 {
			t.Errorf("expected id %q to appear once in GetDisplays batch, got %d times (ids=%v)", id, count, userProvider.lastCallIDs)
		}
	}
	if len(userProvider.lastCallIDs) != 3 {
		t.Errorf("expected 3 distinct ids (invitee-a, invitee-b, lead1), got %d: %v", len(userProvider.lastCallIDs), userProvider.lastCallIDs)
	}
}
