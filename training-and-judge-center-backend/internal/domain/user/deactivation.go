package user

import "time"

type DeactivationStatus string

const (
	DeactivationStatusPending   DeactivationStatus = "PENDING"
	DeactivationStatusConfirmed DeactivationStatus = "CONFIRMED"
	DeactivationStatusExpired   DeactivationStatus = "EXPIRED"
	DeactivationStatusBlocked   DeactivationStatus = "BLOCKED"
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

func (r *DeactivationRequest) ID() string                     { return r.id }
func (r *DeactivationRequest) UserID() string                 { return r.userID }
func (r *DeactivationRequest) VerificationCode() string       { return r.verificationCode }
func (r *DeactivationRequest) ExpiresAt() time.Time           { return r.expiresAt }
func (r *DeactivationRequest) Attempts() int                  { return r.attempts }
func (r *DeactivationRequest) BlockedUntil() *time.Time {
	if r.blockedUntil == nil {
		return nil
	}
	t := *r.blockedUntil
	return &t
}
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
	if r.attempts >= MaxDeactivationAttempts {
		r.status = DeactivationStatusBlocked
		blockedUntil := time.Now().Add(DeactivationBlockDuration)
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

