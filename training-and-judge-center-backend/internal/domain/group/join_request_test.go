package group_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validJoinRequest(t *testing.T) *group.JoinRequest {
	t.Helper()
	req, err := group.NewJoinRequest("id-1", "g-1", shared.RestoreUserID("user-1"), nil, time.Now())
	if err != nil {
		t.Fatalf("NewJoinRequest: %v", err)
	}
	return req
}

func TestNewJoinRequest_EmptyID(t *testing.T) {
	_, err := group.NewJoinRequest("", "g-1", shared.RestoreUserID("user-1"), nil, time.Now())
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestNewJoinRequest_EmptyGroupID(t *testing.T) {
	_, err := group.NewJoinRequest("id-1", "", shared.RestoreUserID("user-1"), nil, time.Now())
	if err == nil {
		t.Fatal("expected error for empty groupID")
	}
}

func TestNewJoinRequest_EmptyRequesterUserID(t *testing.T) {
	_, err := group.NewJoinRequest("id-1", "g-1", shared.RestoreUserID(""), nil, time.Now())
	if err == nil {
		t.Fatal("expected error for empty requesterUserID")
	}
}

func TestNewJoinRequest_StartsAsPending(t *testing.T) {
	req := validJoinRequest(t)
	if req.Status() != group.JoinRequestStatusPending {
		t.Errorf("expected PENDING, got %s", req.Status())
	}
}

func TestApprove_PendingBecomesApproved(t *testing.T) {
	req := validJoinRequest(t)
	if err := req.Approve(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Status() != group.JoinRequestStatusApproved {
		t.Errorf("expected APPROVED, got %s", req.Status())
	}
}

func TestApprove_AlreadyApprovedReturnsError(t *testing.T) {
	req := validJoinRequest(t)
	_ = req.Approve()

	err := req.Approve()
	if err == nil {
		t.Fatal("expected error when approving already-approved request")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeRequestAlreadyProcessed {
		t.Errorf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestApprove_RejectedReturnsError(t *testing.T) {
	req := validJoinRequest(t)
	_ = req.Reject()

	err := req.Approve()
	if err == nil {
		t.Fatal("expected error when approving rejected request")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeRequestAlreadyProcessed {
		t.Errorf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestReject_PendingBecomesRejected(t *testing.T) {
	req := validJoinRequest(t)
	if err := req.Reject(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Status() != group.JoinRequestStatusRejected {
		t.Errorf("expected REJECTED, got %s", req.Status())
	}
}

func TestReject_AlreadyRejectedReturnsError(t *testing.T) {
	req := validJoinRequest(t)
	_ = req.Reject()

	err := req.Reject()
	if err == nil {
		t.Fatal("expected error when rejecting already-rejected request")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeRequestAlreadyProcessed {
		t.Errorf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestReject_ApprovedReturnsError(t *testing.T) {
	req := validJoinRequest(t)
	_ = req.Approve()

	err := req.Reject()
	if err == nil {
		t.Fatal("expected error when rejecting approved request")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != group.ErrCodeRequestAlreadyProcessed {
		t.Errorf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestNewJoinRequestStatus_ValidValues(t *testing.T) {
	for _, s := range []string{"PENDING", "APPROVED", "REJECTED"} {
		if _, err := group.NewJoinRequestStatus(s); err != nil {
			t.Errorf("unexpected error for valid status %q: %v", s, err)
		}
	}
}

func TestNewJoinRequestStatus_InvalidValue(t *testing.T) {
	_, err := group.NewJoinRequestStatus("INVALID")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}
