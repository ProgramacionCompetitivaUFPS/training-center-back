package user

import "context"

// GlobalGroupJoiner adds a newly created user to the platform's global
// group. It is its own port (not domain/group's MemberRepository) because
// application/user must not import domain/group — each domain writes what
// it needs to another domain's tables through its own port and adapter.
type GlobalGroupJoiner interface {
	AddToGlobalGroup(ctx context.Context, userID string) error
}
