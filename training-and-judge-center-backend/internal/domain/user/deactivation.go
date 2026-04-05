package user

import (
	"context"
	"time"
)

type DeactivationStatus string

const (
	DeactivationStatusPending   DeactivationStatus = "PENDING"
	DeactivationStatusConfirmed DeactivationStatus = "CONFIRMED"
	DeactivationStatusExpired   DeactivationStatus = "EXPIRED"
	DeactivationStatusBlocked   DeactivationStatus = "BLOCKED"
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

func RestoreDeactivationRequest(id, userID, verificationCode string, expiresAt time.Time, attempts int, blockedUntil *time.Time, status DeactivationStatus, createdAt, updatedAt time.Time) *DeactivationRequest {
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

func (r *DeactivationRequest) ID() string                     { return r.id }
func (r *DeactivationRequest) UserID() string                 { return r.userID }
func (r *DeactivationRequest) VerificationCode() string       { return r.verificationCode }
func (r *DeactivationRequest) ExpiresAt() time.Time           { return r.expiresAt }
func (r *DeactivationRequest) Attempts() int                  { return r.attempts }
func (r *DeactivationRequest) BlockedUntil() *time.Time       { return r.blockedUntil }
func (r *DeactivationRequest) Status() DeactivationStatus     { return r.status }
func (r *DeactivationRequest) CreatedAt() time.Time           { return r.createdAt }
func (r *DeactivationRequest) UpdatedAt() time.Time           { return r.updatedAt }

func (r *DeactivationRequest) MarkAsExpired() {
	r.status = DeactivationStatusExpired
	r.updatedAt = time.Now()
}

func (r *DeactivationRequest) RegisterFailure() {
	r.attempts++
	r.updatedAt = time.Now()
	if r.attempts >= 5 {
		r.status = DeactivationStatusBlocked
		blockedUntil := time.Now().Add(time.Hour)
		r.blockedUntil = &blockedUntil
	}
}

func (r *DeactivationRequest) IsBlocked() bool {
	return r.status == DeactivationStatusBlocked
}

func (r *DeactivationRequest) Confirm() {
	r.status = DeactivationStatusConfirmed
	r.updatedAt = time.Now()
}

type DeactivationAuditLog struct {
	id               string
	userID           string
	originalEmail    string
	originalNickname string
	occurredAt       time.Time
	ip               *string
	userAgent        *string
}

func RestoreDeactivationAuditLog(id, userID, originalEmail, originalNickname string, occurredAt time.Time, ip, userAgent *string) *DeactivationAuditLog {
	return &DeactivationAuditLog{
		id:               id,
		userID:           userID,
		originalEmail:    originalEmail,
		originalNickname: originalNickname,
		occurredAt:       occurredAt,
		ip:               ip,
		userAgent:        userAgent,
	}
}

func (l *DeactivationAuditLog) ID() string               { return l.id }
func (l *DeactivationAuditLog) UserID() string           { return l.userID }
func (l *DeactivationAuditLog) OriginalEmail() string    { return l.originalEmail }
func (l *DeactivationAuditLog) OriginalNickname() string { return l.originalNickname }
func (l *DeactivationAuditLog) OccurredAt() time.Time    { return l.occurredAt }
func (l *DeactivationAuditLog) IP() *string              { return l.ip }
func (l *DeactivationAuditLog) UserAgent() *string       { return l.userAgent }

type DeactivationRequestRepository interface {
	Save(ctx context.Context, req *DeactivationRequest) error
	FindPendingByUserID(ctx context.Context, userID string) (*DeactivationRequest, error)
	FindByID(ctx context.Context, id string) (*DeactivationRequest, error)
	Update(ctx context.Context, req *DeactivationRequest) error
	InvalidatePendingByUserID(ctx context.Context, userID string, now time.Time) error
}

type DeactivationAuditLogRepository interface {
	Save(ctx context.Context, log *DeactivationAuditLog) error
}
