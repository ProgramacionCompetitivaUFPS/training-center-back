package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestResolveMemberNicknames_DedupesRepeatedNicknames(t *testing.T) {
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "u1", Nickname: "alice", SystemRole: shared.RoleContestant.String()},
	}}

	out, err := resolveMemberNicknames(context.Background(), resolver, []string{"alice", "alice", "alice"}, domainGroup.MemberRoleMember)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 resolved entry after dedup, got %d", len(out))
	}
}

func TestResolveMemberNicknames_DedupesCaseInsensitively(t *testing.T) {
	// The real NicknameResolver adapter resolves case-insensitively (nicknames
	// are stored lowercase), so "Alice" and "alice" both resolve to the same
	// user here — mirroring that behavior to prove the dedup step itself is
	// also case-insensitive, not just resolving to the same result twice.
	alice := &UserDisplay{ID: "u1", Nickname: "alice", SystemRole: shared.RoleContestant.String()}
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"Alice": alice,
		"alice": alice,
	}}

	out, err := resolveMemberNicknames(context.Background(), resolver, []string{"Alice", "alice"}, domainGroup.MemberRoleMember)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected differently-cased duplicates to dedup to 1 entry, got %d", len(out))
	}
}

func TestResolveMemberNicknames_NotFoundReturnsError(t *testing.T) {
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{}}

	_, err := resolveMemberNicknames(context.Background(), resolver, []string{"ghost"}, domainGroup.MemberRoleMember)

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeNicknameNotFound {
		t.Fatalf("expected NICKNAME_NOT_FOUND, got %v", err)
	}
}

func TestResolveMemberNicknames_ContestantAsLeadRejected(t *testing.T) {
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"contestant1": {ID: "u1", Nickname: "contestant1", SystemRole: shared.RoleContestant.String()},
	}}

	_, err := resolveMemberNicknames(context.Background(), resolver, []string{"contestant1"}, domainGroup.MemberRoleLead)

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvalidLeadAssignment {
		t.Fatalf("expected INVALID_LEAD_ASSIGNMENT, got %v", err)
	}
}

func TestResolveMemberNicknames_ContestantAsMemberAllowed(t *testing.T) {
	resolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"contestant1": {ID: "u1", Nickname: "contestant1", SystemRole: shared.RoleContestant.String()},
	}}

	out, err := resolveMemberNicknames(context.Background(), resolver, []string{"contestant1"}, domainGroup.MemberRoleMember)
	if err != nil {
		t.Fatalf("unexpected error: a contestant must be allowed as a plain member: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 resolved entry, got %d", len(out))
	}
}

func TestExcludeUserID_DropsMatchingEntry(t *testing.T) {
	entries := []resolvedNickname{
		{userID: shared.RestoreUserID("u1")},
		{userID: shared.RestoreUserID("u2")},
	}

	out := excludeUserID(entries, shared.RestoreUserID("u1"))

	if len(out) != 1 || out[0].userID.Value() != "u2" {
		t.Fatalf("expected only u2 to remain, got %+v", out)
	}
}

func TestExcludeByUserID_LeadsWinOverMembers(t *testing.T) {
	members := []resolvedNickname{
		{userID: shared.RestoreUserID("u1")},
		{userID: shared.RestoreUserID("u2")},
	}
	leads := []resolvedNickname{
		{userID: shared.RestoreUserID("u1")},
	}

	out := excludeByUserID(members, leads)

	if len(out) != 1 || out[0].userID.Value() != "u2" {
		t.Fatalf("expected only u2 to remain in members, got %+v", out)
	}
}
