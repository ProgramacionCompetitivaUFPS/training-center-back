package group

import (
	"context"
	"strings"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type resolvedNickname struct {
	userID  shared.UserID
	display *UserDisplay
}

// resolveMemberNicknames dedupes repeated nicknames within nicknames, resolves
// each via nicknameResolver, and — when role is MemberRoleLead — rejects
// contestants. It fails fast on the first invalid nickname (not found, or a
// contestant assigned as lead) rather than batching every bad entry.
//
// Dedup keys on the lowercased nickname, matching NicknameResolver's own
// case-insensitive lookup (nicknames are stored lowercase, see
// domain/user/nickname.go) — otherwise "Alice" and "alice" would both
// resolve to the same user and collide on the group_members unique
// constraint, rolling back the whole caller's transaction.
func resolveMemberNicknames(ctx context.Context, nicknameResolver NicknameResolver, nicknames []string, role domainGroup.MemberRole) ([]resolvedNickname, error) {
	seen := make(map[string]struct{}, len(nicknames))
	out := make([]resolvedNickname, 0, len(nicknames))
	for _, nickname := range nicknames {
		key := strings.ToLower(nickname)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}

		target, err := nicknameResolver.ResolveByNickname(ctx, nickname)
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, apperror.NewNotFound(ErrCodeNicknameNotFound, "no user found with nickname: "+nickname)
		}
		if err := requireNotContestantForLead(role, target); err != nil {
			return nil, err
		}
		out = append(out, resolvedNickname{userID: shared.RestoreUserID(target.ID), display: target})
	}
	return out, nil
}

// excludeUserID drops any entry whose userID matches excludeID.
func excludeUserID(entries []resolvedNickname, excludeID shared.UserID) []resolvedNickname {
	out := make([]resolvedNickname, 0, len(entries))
	for _, e := range entries {
		if e.userID.Value() != excludeID.Value() {
			out = append(out, e)
		}
	}
	return out
}

// excludeByUserID drops from members any entry whose userID is also present in leads.
func excludeByUserID(members, leads []resolvedNickname) []resolvedNickname {
	leadIDs := make(map[string]struct{}, len(leads))
	for _, l := range leads {
		leadIDs[l.userID.Value()] = struct{}{}
	}
	out := make([]resolvedNickname, 0, len(members))
	for _, m := range members {
		if _, isLead := leadIDs[m.userID.Value()]; !isLead {
			out = append(out, m)
		}
	}
	return out
}
