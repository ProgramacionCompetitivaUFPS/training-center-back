package group

import "context"

type NicknameResolver interface {
	// ResolveByNickname looks up a user by their unique nickname.
	// Returns (nil, nil) when no user exists with that nickname.
	ResolveByNickname(ctx context.Context, nickname string) (*UserDisplay, error)
}
