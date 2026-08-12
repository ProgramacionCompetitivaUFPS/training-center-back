package user

import (
	"context"
	"time"

	"github.com/google/uuid"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const refreshUnauthorizedMessage = "Invalid or expired session. Please log in again"

const (
	refreshUserRateLimitMaxAttempts = 20
	refreshUserRateLimitWindow      = 10 * time.Minute
)

type RefreshInput struct {
	RefreshToken string
	UserAgent    *string
	IPAddress    *string
}

type RefreshOutput struct {
	Token            string
	RefreshToken     string
	SessionExpiresAt time.Time
}

type RefreshUseCase struct {
	refreshTokenRepo  user.RefreshTokenRepository
	userRepo          user.Repository
	tokenService      user.TokenService
	refreshTokenCodec RefreshTokenCodec
	rateLimiter       appshared.RateLimiter
	rotationCache     RotationCache
}

func NewRefreshUseCase(refreshTokenRepo user.RefreshTokenRepository, userRepo user.Repository, tokenService user.TokenService, refreshTokenCodec RefreshTokenCodec, rateLimiter appshared.RateLimiter, rotationCache RotationCache) *RefreshUseCase {
	return &RefreshUseCase{
		refreshTokenRepo:  refreshTokenRepo,
		userRepo:          userRepo,
		tokenService:      tokenService,
		refreshTokenCodec: refreshTokenCodec,
		rateLimiter:       rateLimiter,
		rotationCache:     rotationCache,
	}
}

func (uc *RefreshUseCase) Execute(ctx context.Context, in RefreshInput) (*RefreshOutput, error) {
	now := time.Now()

	if in.RefreshToken == "" {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	plainSecret, userID, err := uc.refreshTokenCodec.Unwrap(in.RefreshToken)
	if err != nil {
		// invalid signature / malformed envelope → reject before ever touching the DB. Discard
		// the codec's own error so every refresh 401 reads identically — the reason never leaks.
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	if err := uc.checkRateLimit(ctx, "rate_limit:refresh:user:"+userID, refreshUserRateLimitMaxAttempts, refreshUserRateLimitWindow); err != nil {
		return nil, err
	}

	hash := hashRefreshTokenSecret(plainSecret)

	token, err := uc.refreshTokenRepo.FindByTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	if token == nil {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	foundUser, err := uc.userRepo.FindByID(ctx, token.UserID())
	if err != nil {
		return nil, err
	}
	if foundUser == nil || foundUser.Status() != user.StatusActive {
		return nil, apperror.NewForbidden(ErrCodeAccountDeactivated, "This account has been deactivated")
	}

	if token.IsRevoked() {
		return uc.handleAlreadyRevoked(ctx, token.TokenHash(), token.FamilyID(), now, foundUser)
	}

	if token.IsExpired(now) {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	newSecret, err := generateRefreshTokenSecret()
	if err != nil {
		return nil, apperror.NewInternal()
	}

	successor, err := token.Successor(uuid.New().String(), hashRefreshTokenSecret(newSecret), in.UserAgent, in.IPAddress, now)
	if err != nil {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	accessToken, err := uc.tokenService.GenerateToken(ctx, foundUser, token.FamilyID())
	if err != nil {
		return nil, err
	}

	wrapped, err := uc.refreshTokenCodec.Wrap(ctx, newSecret, foundUser.ID())
	if err != nil {
		return nil, err
	}

	rotated, err := uc.refreshTokenRepo.Rotate(ctx, token.TokenHash(), successor)
	if err != nil {
		return nil, err
	}
	if !rotated {
		return uc.handleAlreadyRevoked(ctx, token.TokenHash(), token.FamilyID(), now, foundUser)
	}

	out := &RefreshOutput{
		Token:            accessToken,
		RefreshToken:     wrapped,
		SessionExpiresAt: successor.AbsoluteExpiresAt(),
	}

	if err := uc.rotationCache.Save(ctx, token.TokenHash(), *out, user.RefreshGraceWindow); err != nil {
		_ = err // best-effort: cache miss just means no replay credential to hand back
	}

	return out, nil
}

func (uc *RefreshUseCase) handleAlreadyRevoked(ctx context.Context, tokenHash, familyID string, now time.Time, foundUser *user.User) (*RefreshOutput, error) {
	if cached, err := uc.rotationCache.Get(ctx, tokenHash); err == nil && cached != nil {
		return cached, nil
	}

	current, err := uc.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if current == nil || !current.IsRevoked() {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	if !current.WithinGraceWindow(now) {
		if err := uc.refreshTokenRepo.RevokeByFamilyID(ctx, familyID, now); err != nil {
			return nil, err
		}
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	active, err := uc.refreshTokenRepo.FindActiveByFamilyID(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if active == nil {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, refreshUnauthorizedMessage)
	}

	accessToken, err := uc.tokenService.GenerateToken(ctx, foundUser, familyID)
	if err != nil {
		return nil, err
	}

	return &RefreshOutput{
		Token:            accessToken,
		RefreshToken:     "",
		SessionExpiresAt: active.AbsoluteExpiresAt(),
	}, nil
}

func (uc *RefreshUseCase) checkRateLimit(ctx context.Context, key string, maxAttempts int, window time.Duration) error {
	allowed, err := uc.rateLimiter.Allow(ctx, key, maxAttempts, window)
	if err != nil {
		return err
	}
	if !allowed {
		return apperror.NewTooManyRequests(ErrCodeTooManyRequests,
			"Too many refresh attempts. Please try again later.", int(window.Seconds()))
	}
	return nil
}
