package user

import "context"

type OAuthIdentityRepository interface {
	Save(ctx context.Context, identity *OAuthIdentity) error
	FindByProvider(ctx context.Context, provider OAuthProvider, providerUserID string) (*OAuthIdentity, error)
}
