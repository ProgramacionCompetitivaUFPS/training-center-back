package user

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type OAuthProvider string

const OAuthProviderGoogle OAuthProvider = "GOOGLE"

type OAuthIdentity struct {
	id             string
	userID         string
	provider       OAuthProvider
	providerUserID string
	createdAt      time.Time
}

func NewOAuthIdentity(id, userID string, provider OAuthProvider, providerUserID string, now time.Time) (*OAuthIdentity, error) {
	if id == "" || userID == "" || providerUserID == "" {
		return nil, apperror.NewInternal()
	}
	return &OAuthIdentity{
		id:             id,
		userID:         userID,
		provider:       provider,
		providerUserID: providerUserID,
		createdAt:      now.UTC(),
	}, nil
}

func RestoreOAuthIdentity(id, userID string, provider OAuthProvider, providerUserID string, createdAt time.Time) *OAuthIdentity {
	return &OAuthIdentity{
		id:             id,
		userID:         userID,
		provider:       provider,
		providerUserID: providerUserID,
		createdAt:      createdAt,
	}
}

func (o *OAuthIdentity) ID() string              { return o.id }
func (o *OAuthIdentity) UserID() string          { return o.userID }
func (o *OAuthIdentity) Provider() OAuthProvider { return o.provider }
func (o *OAuthIdentity) ProviderUserID() string  { return o.providerUserID }
func (o *OAuthIdentity) CreatedAt() time.Time    { return o.createdAt }
