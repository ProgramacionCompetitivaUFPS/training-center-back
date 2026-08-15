package user

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LoginInput struct {
	Email           string
	Password        string
	RememberSession bool
	UserAgent       *string
	IPAddress       *string
}

type LoginOutput = SessionOutput

type LoginUseCase struct {
	repo              user.Repository
	refreshTokenRepo  user.RefreshTokenRepository
	tokenService      user.TokenService
	refreshTokenCodec RefreshTokenCodec
	rateLimiter       appshared.RateLimiter
}

func NewLoginUseCase(repo user.Repository, refreshTokenRepo user.RefreshTokenRepository, tokenService user.TokenService, refreshTokenCodec RefreshTokenCodec, rateLimiter appshared.RateLimiter) *LoginUseCase {
	return &LoginUseCase{repo: repo, refreshTokenRepo: refreshTokenRepo, tokenService: tokenService, refreshTokenCodec: refreshTokenCodec, rateLimiter: rateLimiter}
}

func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	var fieldErrors []apperror.FieldError

	if input.Email == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "email", Message: "Email is required"})
	}
	if input.Password == "" {
		fieldErrors = append(fieldErrors, apperror.FieldError{Field: "password", Message: "Password is required"})
	}
	if len(fieldErrors) > 0 {
		return nil, apperror.NewValidation(fieldErrors)
	}

	rateKey := "rate_limit:login:" + input.Email
	allowed, err := uc.rateLimiter.Allow(ctx, rateKey, 10, 15*time.Minute)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, apperror.NewTooManyRequests(ErrCodeTooManyRequests,
			"Too many login attempts. Please try again in 15 minutes.", 900)
	}

	email, err := user.NewEmail(input.Email)
	if err != nil {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidCredentials, "Invalid email or password")
	}

	foundUser, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if foundUser == nil {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidCredentials, "Invalid email or password")
	}

	if foundUser.Status() != user.StatusActive {
		return nil, apperror.NewForbidden(ErrCodeAccountDeactivated, "This account has been deactivated")
	}

	if !foundUser.Password().Compare(input.Password) {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidCredentials, "Invalid email or password")
	}

	session, err := issueSession(ctx, foundUser, input.RememberSession, input.UserAgent, input.IPAddress,
		uc.tokenService, uc.refreshTokenRepo, uc.refreshTokenCodec, time.Now())
	if err != nil {
		return nil, err
	}

	// Login successful: reset rate limit so the user has fresh attempts.
	if err := uc.rateLimiter.Reset(ctx, rateKey); err != nil {
		// Don't fail the login; the token was already generated.
		_ = err
	}

	return &LoginOutput{
		Token:            session.Token,
		RefreshToken:     session.RefreshToken,
		SessionExpiresAt: session.SessionExpiresAt,
		User:             userToDTO(foundUser),
	}, nil
}
