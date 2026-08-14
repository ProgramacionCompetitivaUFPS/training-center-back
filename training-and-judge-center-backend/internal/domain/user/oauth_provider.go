package user

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	oauthProviderGoogle = "GOOGLE"
)

type OAuthProvider struct{ value string }

var OAuthProviderGoogle = OAuthProvider{value: oauthProviderGoogle}

func NewOAuthProvider(value string) (OAuthProvider, error) {
	switch value {
	case oauthProviderGoogle:
		return OAuthProvider{value: value}, nil
	default:
		return OAuthProvider{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "provider", Message: "invalid provider: " + value},
		})
	}
}

func RestoreOAuthProvider(value string) OAuthProvider {
	return OAuthProvider{value: value}
}

func (p OAuthProvider) String() string {
	return p.value
}
