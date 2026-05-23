package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRejectRequestUseCase(memberRepo *mockMemberRepository, reqRepo *mockJoinRequestRepository) *RejectRequestUseCase {
	return NewRejectRequestUseCase(memberRepo, reqRepo)
}

func TestRejectRequest_NonLeadReturns403(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectRequestUseCase(&mockMemberRepository{}, reqRepo)

	_, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: asContestant("nobody"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestRejectRequest_AdminCanRejectWithoutMembership(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectRequestUseCase(&mockMemberRepository{}, reqRepo)

	out, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: asAdmin("admin-id"),
	})
	if err != nil {
		t.Fatalf("admin should be able to reject, got: %v", err)
	}
	if out.Request.Status != domainGroup.JoinRequestStatusRejected.String() {
		t.Errorf("expected REJECTED, got %s", out.Request.Status)
	}
}

func TestRejectRequest_RequestNotFoundReturns404(t *testing.T) {
	uc := newRejectRequestUseCase(leadMemberRepo("g1", "lead-id"), &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "nonexistent",
		CurrentUser: asContestant("lead-id"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND, got %v", err)
	}
}

func TestRejectRequest_AlreadyProcessedReturns400(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	_ = req.Approve()
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectRequestUseCase(leadMemberRepo("g1", "lead-id"), reqRepo)

	_, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: asContestant("lead-id"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestAlreadyProcessed {
		t.Fatalf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestRejectRequest_SuccessUpdatesStatus(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectRequestUseCase(leadMemberRepo("g1", "lead-id"), reqRepo)

	out, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: asContestant("lead-id"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request.Status != domainGroup.JoinRequestStatusRejected.String() {
		t.Errorf("expected REJECTED, got %s", out.Request.Status)
	}
	if len(reqRepo.savedRequests) == 0 {
		t.Error("expected request to be saved")
	}
}
