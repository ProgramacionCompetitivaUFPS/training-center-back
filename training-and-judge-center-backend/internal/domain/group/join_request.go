package group

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type JoinRequest struct {
	id              string
	groupID         string
	requesterUserID shared.UserID
	message         *string
	status          JoinRequestStatus
	createdAt       time.Time
}

func NewJoinRequest(id, groupID string, requesterUserID shared.UserID, message *string, clock func() time.Time) (*JoinRequest, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	if groupID == "" {
		return nil, apperror.NewInternal()
	}
	if requesterUserID.Value() == "" {
		return nil, apperror.NewInternal()
	}
	if clock == nil {
		clock = time.Now
	}
	return &JoinRequest{
		id:              id,
		groupID:         groupID,
		requesterUserID: requesterUserID,
		message:         message,
		status:          JoinRequestStatusPending,
		createdAt:       clock().UTC(),
	}, nil
}

func RestoreJoinRequest(id, groupID string, requesterUserID shared.UserID, message *string, status JoinRequestStatus, createdAt time.Time) *JoinRequest {
	return &JoinRequest{
		id:              id,
		groupID:         groupID,
		requesterUserID: requesterUserID,
		message:         message,
		status:          status,
		createdAt:       createdAt,
	}
}

func (r *JoinRequest) ID() string                     { return r.id }
func (r *JoinRequest) GroupID() string                { return r.groupID }
func (r *JoinRequest) RequesterUserID() shared.UserID { return r.requesterUserID }
func (r *JoinRequest) Message() *string               { return r.message }
func (r *JoinRequest) Status() JoinRequestStatus      { return r.status }
func (r *JoinRequest) CreatedAt() time.Time           { return r.createdAt }
func (r *JoinRequest) IsPending() bool                { return r.status == JoinRequestStatusPending }

func (r *JoinRequest) Approve() error {
	if !r.IsPending() {
		return apperror.NewBadRequest(ErrCodeRequestAlreadyProcessed, "this request has already been processed")
	}
	r.status = JoinRequestStatusApproved
	return nil
}

func (r *JoinRequest) Reject() error {
	if !r.IsPending() {
		return apperror.NewBadRequest(ErrCodeRequestAlreadyProcessed, "this request has already been processed")
	}
	r.status = JoinRequestStatusRejected
	return nil
}
