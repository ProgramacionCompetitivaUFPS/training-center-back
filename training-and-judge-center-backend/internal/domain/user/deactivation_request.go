package user

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	MaxDeactivationAttempts   = 5
	DeactivationBlockDuration = time.Hour
)

type DeactivationRequest struct {
	id               string
	userID           string
	verificationCode string
	expiresAt        time.Time
	attempts         int
	blockedUntil     *time.Time
	status           DeactivationStatus
	createdAt        time.Time
	updatedAt        time.Time
}

func NewDeactivationRequest(id, userID, verificationCode string, now time.Time) (*DeactivationRequest, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	t := now.UTC()
	return &DeactivationRequest{
		id:               id,
		userID:           userID,
		verificationCode: verificationCode,
		expiresAt:        t.Add(RequestExpiryDuration),
		attempts:         0,
		status:           DeactivationStatusPending,
		createdAt:        t,
		updatedAt:        t,
	}, nil
}

func RestoreDeactivationRequest(id, userID, verificationCode string, expiresAt time.Time, attempts int, blockedUntil *time.Time, status DeactivationStatus, createdAt, updatedAt time.Time) *DeactivationRequest {
	if status == DeactivationStatusBlocked && blockedUntil == nil {
		status = DeactivationStatusExpired
	}
	return &DeactivationRequest{
		id:               id,
		userID:           userID,
		verificationCode: verificationCode,
		expiresAt:        expiresAt,
		attempts:         attempts,
		blockedUntil:     blockedUntil,
		status:           status,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}

func (r *DeactivationRequest) ID() string               { return r.id }
func (r *DeactivationRequest) UserID() string           { return r.userID }
func (r *DeactivationRequest) VerificationCode() string { return r.verificationCode }
func (r *DeactivationRequest) ExpiresAt() time.Time     { return r.expiresAt }
func (r *DeactivationRequest) Attempts() int            { return r.attempts }
func (r *DeactivationRequest) BlockedUntil() *time.Time {
	if r.blockedUntil == nil {
		return nil
	}
	t := *r.blockedUntil
	return &t
}
func (r *DeactivationRequest) Status() DeactivationStatus { return r.status }
func (r *DeactivationRequest) CreatedAt() time.Time       { return r.createdAt }
func (r *DeactivationRequest) UpdatedAt() time.Time       { return r.updatedAt }

func (r *DeactivationRequest) MarkAsExpired(now time.Time) {
	r.status = DeactivationStatusExpired
	r.updatedAt = now.UTC()
}

func (r *DeactivationRequest) RegisterFailure(now time.Time) {
	t := now.UTC()
	r.attempts++
	r.updatedAt = t
	if r.attempts >= MaxDeactivationAttempts {
		r.status = DeactivationStatusBlocked
		blockedUntil := t.Add(DeactivationBlockDuration)
		r.blockedUntil = &blockedUntil
	}
}

func (r *DeactivationRequest) IsBlocked() bool {
	return r.status == DeactivationStatusBlocked
}

func (r *DeactivationRequest) IsCurrentlyBlocked(now time.Time) bool {
	return r.status == DeactivationStatusBlocked &&
		r.blockedUntil != nil &&
		now.Before(*r.blockedUntil)
}

func (r *DeactivationRequest) IsExpired(now time.Time) bool {
	return now.After(r.expiresAt) || r.status == DeactivationStatusExpired
}

func (r *DeactivationRequest) Confirm(now time.Time) {
	r.status = DeactivationStatusConfirmed
	r.updatedAt = now.UTC()
}
