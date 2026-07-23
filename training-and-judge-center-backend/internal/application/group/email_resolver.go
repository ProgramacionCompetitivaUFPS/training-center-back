package group

import "context"

type EmailResolver interface {
	// ResolveByEmail looks up a user by their unique email address.
	// Returns (nil, nil) when no user exists with that email.
	ResolveByEmail(ctx context.Context, email string) (*UserDisplay, error)
}
