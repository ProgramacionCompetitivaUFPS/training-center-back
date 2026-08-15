package user

import "context"

type GoogleClaims struct {
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
}

type GoogleIDTokenVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleClaims, error)
}
