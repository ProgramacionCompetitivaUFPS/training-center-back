package auth

import (
	"context"
	"log/slog"

	"google.golang.org/api/idtoken"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type validateFunc func(ctx context.Context, idToken, audience string) (*idtoken.Payload, error)

var _ appuser.GoogleIDTokenVerifier = (*GoogleVerifier)(nil)

type GoogleVerifier struct {
	clientID string
	validate validateFunc
}

func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{clientID: clientID, validate: idtoken.Validate}
}

func (v *GoogleVerifier) Verify(ctx context.Context, idTokenStr string) (*appuser.GoogleClaims, error) {
	payload, err := v.validate(ctx, idTokenStr, v.clientID)
	if err != nil {
		slog.ErrorContext(ctx, "google id token validation failed", "error", err)
		return nil, apperror.NewUnauthorized(appuser.ErrCodeInvalidGoogleToken, "invalid Google ID token")
	}

	email, _ := payload.Claims["email"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)
	name, _ := payload.Claims["name"].(string)

	return &appuser.GoogleClaims{
		Sub:           payload.Subject,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
	}, nil
}
