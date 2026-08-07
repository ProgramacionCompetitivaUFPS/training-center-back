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

func NewRefreshToken(id, userID, familyID, tokenHash string, userAgent, ipAddress *string, rememberSession bool, now time.Time) (*RefreshToken, error) {
	ceiling := DefaultSessionCeiling
	if rememberSession {
		ceiling = RememberSessionCeiling
	}
	return newRefreshTokenWithCeiling(id, userID, familyID, tokenHash, userAgent, ipAddress, now.UTC().Add(ceiling), now)
}

// newRefreshTokenWithCeiling is the only place that accepts an already-computed
// absoluteExpiresAt. It is unexported so that application code can never construct a
// token with an arbitrary ceiling — the only ways in are NewRefreshToken (login, derives
// the ceiling from rememberSession) and Rotate (inherits the ceiling from the token it
// replaces).
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
		userAgent:         userAgent,
		ipAddress:         ipAddress,
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
		revokedAt:         revokedAt,
		replacedByID:      replacedByID,
		userAgent:         userAgent,
		ipAddress:         ipAddress,
	}
}

func (t *RefreshToken) ID() string                   { return t.id }
func (t *RefreshToken) UserID() string               { return t.userID }
func (t *RefreshToken) FamilyID() string             { return t.familyID }
func (t *RefreshToken) TokenHash() string            { return t.tokenHash }
func (t *RefreshToken) IssuedAt() time.Time          { return t.issuedAt }
func (t *RefreshToken) AbsoluteExpiresAt() time.Time { return t.absoluteExpiresAt }
func (t *RefreshToken) UserAgent() *string           { return t.userAgent }
func (t *RefreshToken) IPAddress() *string           { return t.ipAddress }

func (t *RefreshToken) RevokedAt() *time.Time {
	if t.revokedAt == nil {
		return nil
	}
	revokedAt := *t.revokedAt
	return &revokedAt
}

func (t *RefreshToken) ReplacedByID() *string {
	if t.replacedByID == nil {
		return nil
	}
	replacedByID := *t.replacedByID
	return &replacedByID
}

func (t *RefreshToken) IsRevoked() bool { return t.revokedAt != nil }

func (t *RefreshToken) IsExpired(now time.Time) bool { return now.After(t.absoluteExpiresAt) }

// WithinGraceWindow reports whether a replay of this already-revoked token should be
// tolerated as a benign multi-tab race instead of treated as reuse/theft.
func (t *RefreshToken) WithinGraceWindow(now time.Time) bool {
	return t.revokedAt != nil && now.Sub(*t.revokedAt) <= RefreshGraceWindow
}

func (t *RefreshToken) Revoke(now time.Time, replacedByID *string) {
	revokedAt := now.UTC()
	t.revokedAt = &revokedAt
	t.replacedByID = replacedByID
}

// Rotate builds the successor of an active token, inheriting its family and absolute
// ceiling. It does not mutate the receiver — whether the receiver actually ends up
// revoked is decided by the atomic persistence operation (RefreshTokenRepository.Rotate),
// not by this call, so marking it here in advance would risk lying about a race this
// method cannot see the outcome of.
//
// The IsRevoked() check below is a local fail-fast for the obvious case (a use case bug
// calling Rotate twice on the same in-memory object) — it is not the security guarantee
// against reuse/theft across concurrent requests. That guarantee is the atomic, DB-level
// conditional UPDATE behind RefreshTokenRepository.Rotate.
func (t *RefreshToken) Rotate(newID, tokenHash string, userAgent, ipAddress *string, now time.Time) (*RefreshToken, error) {
	if t.IsRevoked() {
		return nil, apperror.NewConflict(ErrCodeRefreshTokenAlreadyRevoked, "refresh token is already revoked")
	}
	return newRefreshTokenWithCeiling(newID, t.userID, t.familyID, tokenHash, userAgent, ipAddress, t.absoluteExpiresAt, now)
}
