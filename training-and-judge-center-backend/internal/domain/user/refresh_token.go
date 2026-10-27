package user

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	DefaultSessionCeiling  = 24 * time.Hour
	RememberSessionCeiling = 30 * 24 * time.Hour
	RefreshGraceWindow     = 10 * time.Second
)

type RefreshToken struct {
	id                string
	userID            string
	familyID          string
	tokenHash         string
	issuedAt          time.Time
	absoluteExpiresAt time.Time
	revokedAt         *time.Time
	replacedByID      *string
	userAgent         *string
	ipAddress         *string
}

// NewRefreshToken originates a brand-new family (login only) — to continue a session, use Rotate.
func NewRefreshToken(id, userID, familyID, tokenHash string, userAgent, ipAddress *string, rememberSession bool, now time.Time) (*RefreshToken, error) {
	ceiling := DefaultSessionCeiling
	if rememberSession {
		ceiling = RememberSessionCeiling
	}
	return newRefreshTokenWithCeiling(id, userID, familyID, tokenHash, userAgent, ipAddress, now.UTC().Add(ceiling), now)
}

// newRefreshTokenWithCeiling is unexported so callers can't set an arbitrary ceiling.
func newRefreshTokenWithCeiling(id, userID, familyID, tokenHash string, userAgent, ipAddress *string, absoluteExpiresAt, now time.Time) (*RefreshToken, error) {
	if id == "" || userID == "" || familyID == "" || tokenHash == "" {
		return nil, apperror.NewInternal()
	}
	return &RefreshToken{
		id:                id,
		userID:            userID,
		familyID:          familyID,
		tokenHash:         tokenHash,
		issuedAt:          now.UTC(),
		absoluteExpiresAt: absoluteExpiresAt.UTC(),
		userAgent:         copyStringPtr(userAgent),
		ipAddress:         copyStringPtr(ipAddress),
	}, nil
}

func RestoreRefreshToken(id, userID, familyID, tokenHash string, issuedAt, absoluteExpiresAt time.Time, revokedAt *time.Time, replacedByID, userAgent, ipAddress *string) *RefreshToken {
	return &RefreshToken{
		id:                id,
		userID:            userID,
		familyID:          familyID,
		tokenHash:         tokenHash,
		issuedAt:          issuedAt,
		absoluteExpiresAt: absoluteExpiresAt,
		revokedAt:         copyTimePtr(revokedAt),
		replacedByID:      copyStringPtr(replacedByID),
		userAgent:         copyStringPtr(userAgent),
		ipAddress:         copyStringPtr(ipAddress),
	}
}

func (t *RefreshToken) ID() string                   { return t.id }
func (t *RefreshToken) UserID() string               { return t.userID }
func (t *RefreshToken) FamilyID() string             { return t.familyID }
func (t *RefreshToken) TokenHash() string            { return t.tokenHash }
func (t *RefreshToken) IssuedAt() time.Time          { return t.issuedAt }
func (t *RefreshToken) AbsoluteExpiresAt() time.Time { return t.absoluteExpiresAt }
func (t *RefreshToken) UserAgent() *string           { return copyStringPtr(t.userAgent) }
func (t *RefreshToken) IPAddress() *string           { return copyStringPtr(t.ipAddress) }
func (t *RefreshToken) RevokedAt() *time.Time        { return copyTimePtr(t.revokedAt) }
func (t *RefreshToken) ReplacedByID() *string        { return copyStringPtr(t.replacedByID) }

func (t *RefreshToken) IsRevoked() bool { return t.revokedAt != nil }

func (t *RefreshToken) IsExpired(now time.Time) bool { return now.After(t.absoluteExpiresAt) }

func (t *RefreshToken) WithinGraceWindow(now time.Time) bool {
	return t.revokedAt != nil && now.Sub(*t.revokedAt) <= RefreshGraceWindow
}

func (t *RefreshToken) Revoke(now time.Time, replacedByID *string) {
	if t.revokedAt != nil {
		return
	}
	revokedAt := now.UTC()
	t.revokedAt = &revokedAt
	t.replacedByID = copyStringPtr(replacedByID)
}

// Successor derives the next token in the family without mutating the receiver
func (t *RefreshToken) Successor(newID, tokenHash string, userAgent, ipAddress *string, now time.Time) (*RefreshToken, error) {
	if t.IsRevoked() {
		return nil, apperror.NewConflict(ErrCodeRefreshTokenAlreadyRevoked, "refresh token is already revoked")
	}
	if t.IsExpired(now) {
		return nil, apperror.NewConflict(ErrCodeRefreshTokenExpired, "refresh token has passed its absolute session ceiling")
	}
	return newRefreshTokenWithCeiling(newID, t.userID, t.familyID, tokenHash, userAgent, ipAddress, t.absoluteExpiresAt, now)
}

func copyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	c := *s
	return &c
}

func copyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	c := *t
	return &c
}
