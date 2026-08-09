package user

import "context"

type RefreshTokenCodec interface {
	Wrap(ctx context.Context, secret, userID string) (string, error)
	Unwrap(wrapped string) (secret, userID string, err error)
}
