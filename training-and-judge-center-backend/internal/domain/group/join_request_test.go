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
	req, err := group.NewJoinRequest("id-1", "g-1", shared.RestoreUserID("user-1"), nil, testNow)
	if err != nil {
		t.Fatalf("NewJoinRequest: %v", err)
	}
	return req
}

func TestNewJoinRequest_EmptyID(t *testing.T) {
	_, err := group.NewJoinRequest("", "g-1", shared.RestoreUserID("user-1"), nil, testNow)
	if err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestNewJoinRequest_EmptyGroupID(t *testing.T) {
	_, err := group.NewJoinRequest("id-1", "", shared.RestoreUserID("user-1"), nil, testNow)
	if err == nil {
		t.Fatal("expected error for empty groupID")
	}
}

func TestNewJoinRequest_EmptyRequesterUserID(t *testing.T) {
	_, err := group.NewJoinRequest("id-1", "g-1", shared.RestoreUserID(""), nil, testNow)
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

func TestNewJoinRequest_StoresTheFieldsAndTheInstantInUTC(t *testing.T) {
	message := "please let me in"

	req, err := group.NewJoinRequest("req-1", "g-1", shared.RestoreUserID("user-1"), &message, localNow)

	if err != nil {
		t.Fatalf("NewJoinRequest: %v", err)
	}
	if req.ID() != "req-1" || req.GroupID() != "g-1" || req.RequesterUserID().Value() != "user-1" {
		t.Errorf("fields = %s/%s/%s, want req-1/g-1/user-1", req.ID(), req.GroupID(), req.RequesterUserID().Value())
	}
	if req.Message() == nil || *req.Message() != "please let me in" {
		t.Errorf("Message() = %v, want the stored message", req.Message())
	}
	assertInUTC(t, "CreatedAt", req.CreatedAt(), time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))
}

func TestRestoreJoinRequest_KeepsTheStoredValues(t *testing.T) {
	message := "stored"

	req := group.RestoreJoinRequest("req-1", "g-1", shared.RestoreUserID("user-1"), &message, group.JoinRequestStatusRejected, testNow)

	if req.ID() != "req-1" || req.GroupID() != "g-1" || req.RequesterUserID().Value() != "user-1" || req.Status() != group.JoinRequestStatusRejected || !req.CreatedAt().Equal(testNow) {
		t.Errorf("restored request = %s/%s/%s/%s/%v", req.ID(), req.GroupID(), req.RequesterUserID().Value(), req.Status(), req.CreatedAt())
	}
	if req.Message() == nil || *req.Message() != "stored" {
		t.Errorf("Message() = %v, want stored", req.Message())
	}
	if req.IsPending() {
		t.Error("IsPending() = true for a rejected request")
	}
}
