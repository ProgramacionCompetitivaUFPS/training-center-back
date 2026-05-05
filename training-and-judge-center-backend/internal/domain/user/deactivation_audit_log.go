package user

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeactivationAuditLog struct {
	id               string
	userID           string
	originalEmail    string
	originalNickname string
	occurredAt       time.Time
	ip               *string
	userAgent        *string
}

func NewDeactivationAuditLog(id, userID, originalEmail, originalNickname string, occurredAt time.Time, ip, userAgent *string) (*DeactivationAuditLog, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	return &DeactivationAuditLog{
		id:               id,
		userID:           userID,
		originalEmail:    originalEmail,
		originalNickname: originalNickname,
		occurredAt:       occurredAt,
		ip:               ip,
		userAgent:        userAgent,
	}, nil
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
