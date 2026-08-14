package user

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// Google's profile has no country/city/institution; the user fills these in later via PUT /users.
const googleAccountPlaceholder = "N/A"

type LoginWithGoogleUseCase struct {
	userRepo          user.Repository
	oauthIdentityRepo user.OAuthIdentityRepository
	refreshTokenRepo  user.RefreshTokenRepository
	tokenService      user.TokenService
	refreshTokenCodec RefreshTokenCodec
	googleVerifier    GoogleIDTokenVerifier
	txManager         appshared.TransactionManager
}

func NewLoginWithGoogleUseCase(
	userRepo user.Repository,
	oauthIdentityRepo user.OAuthIdentityRepository,
	refreshTokenRepo user.RefreshTokenRepository,
	tokenService user.TokenService,
	refreshTokenCodec RefreshTokenCodec,
	googleVerifier GoogleIDTokenVerifier,
	txManager appshared.TransactionManager,
) *LoginWithGoogleUseCase {
	return &LoginWithGoogleUseCase{
		userRepo:          userRepo,
		oauthIdentityRepo: oauthIdentityRepo,
		refreshTokenRepo:  refreshTokenRepo,
		tokenService:      tokenService,
		refreshTokenCodec: refreshTokenCodec,
		googleVerifier:    googleVerifier,
		txManager:         txManager,
	}
}

type LoginWithGoogleInput struct {
	IDToken         string
	RememberSession bool
	UserAgent       *string
	IPAddress       *string
}

type LoginWithGoogleOutput = SessionOutput

func (uc *LoginWithGoogleUseCase) Execute(ctx context.Context, in LoginWithGoogleInput) (*LoginWithGoogleOutput, error) {
	if in.IDToken == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "id_token", Message: "id_token is required"},
		})
	}

	claims, err := uc.googleVerifier.Verify(ctx, in.IDToken)
	if err != nil {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidGoogleToken, "invalid Google ID token")
	}
	if !claims.EmailVerified {
		return nil, apperror.NewUnauthorized(ErrCodeGoogleEmailNotVerified, "Google account email is not verified")
	}

	now := time.Now()

	foundUser, err := uc.findOrCreateUser(ctx, claims, now)
	if err != nil {
		return nil, err
	}

	if foundUser.Status() != user.StatusActive {
		return nil, apperror.NewForbidden(ErrCodeAccountDeactivated, "This account has been deactivated")
	}

	session, err := issueSession(ctx, foundUser, in.RememberSession, in.UserAgent, in.IPAddress,
		uc.tokenService, uc.refreshTokenRepo, uc.refreshTokenCodec, now)
	if err != nil {
		return nil, err
	}

	return &LoginWithGoogleOutput{
		Token:            session.Token,
		RefreshToken:     session.RefreshToken,
		SessionExpiresAt: session.SessionExpiresAt,
		User:             userToDTO(foundUser),
	}, nil
}

func (uc *LoginWithGoogleUseCase) findOrCreateUser(ctx context.Context, claims *GoogleClaims, now time.Time) (*user.User, error) {
	if existingIdentity, err := uc.oauthIdentityRepo.FindByProvider(ctx, user.OAuthProviderGoogle, claims.Sub); err != nil {
		return nil, err
	} else if existingIdentity != nil {
		return uc.userByIdentity(ctx, existingIdentity)
	}

	email, err := user.NewEmail(claims.Email)
	if err != nil {
		return nil, apperror.NewUnauthorized(ErrCodeInvalidGoogleEmail, "Google account email is invalid")
	}

	foundUser, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	isNewUser := foundUser == nil
	if isNewUser {
		foundUser, err = uc.newUserFromGoogle(claims, email, now)
		if err != nil {
			return nil, err
		}
	}

	identity, err := user.NewOAuthIdentity(uuid.New().String(), foundUser.ID(), user.OAuthProviderGoogle, claims.Sub, now)
	if err != nil {
		return nil, err
	}

	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if isNewUser {
			if err := uc.userRepo.Save(txCtx, foundUser); err != nil {
				return err
			}
		}
		return uc.oauthIdentityRepo.Save(txCtx, identity)
	})
	if err == nil {
		return foundUser, nil
	}

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindConflict {
		return nil, err
	}

	winnerIdentity, findErr := uc.oauthIdentityRepo.FindByProvider(ctx, user.OAuthProviderGoogle, claims.Sub)
	if findErr != nil {
		return nil, findErr
	}
	if winnerIdentity == nil {
		return nil, apperror.NewInternal()
	}
	return uc.userByIdentity(ctx, winnerIdentity)
}

func (uc *LoginWithGoogleUseCase) userByIdentity(ctx context.Context, identity *user.OAuthIdentity) (*user.User, error) {
	foundUser, err := uc.userRepo.FindByID(ctx, identity.UserID())
	if err != nil {
		return nil, err
	}
	if foundUser == nil {
		return nil, apperror.NewInternal()
	}
	return foundUser, nil
}

func (uc *LoginWithGoogleUseCase) newUserFromGoogle(claims *GoogleClaims, email user.Email, now time.Time) (*user.User, error) {
	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = googleEmailLocalPart(claims.Email)
	}

	nickname, err := googleNicknameFromEmail(claims.Email)
	if err != nil {
		return nil, err
	}

	return user.NewUser(
		uuid.New().String(),
		now,
		email,
		user.NewPasswordNone(),
		name,
		nickname,
		googleAccountPlaceholder,
		googleAccountPlaceholder,
		googleAccountPlaceholder,
	)
}

func googleEmailLocalPart(email string) string {
	if idx := strings.IndexByte(email, '@'); idx != -1 {
		return email[:idx]
	}
	return email
}

func googleNicknameFromEmail(email string) (user.Nickname, error) {
	local := strings.ToLower(googleEmailLocalPart(email))

	var b strings.Builder
	for _, r := range local {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	sanitized := b.String()

	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]

	maxLocalLen := 30 - 1 - len(suffix)
	if len(sanitized) > maxLocalLen {
		sanitized = sanitized[:maxLocalLen]
	}

	return user.NewNickname(sanitized + "-" + suffix)
}
