package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRejectUC(memberRepo *fakeMemberRepo, reqRepo *fakeJoinRequestRepo) *RejectRequestUseCase {
	return NewRejectRequestUseCase(memberRepo, reqRepo)
}

func TestRejectRequest_NonLeadReturns403(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectUC(&fakeMemberRepo{}, reqRepo)

	_, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("nobody", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestRejectRequest_AdminCanRejectWithoutMembership(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectUC(&fakeMemberRepo{}, reqRepo)

	out, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("admin-id", shared.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("admin should be able to reject, got: %v", err)
	}
	if out.Request.Status() != domainGroup.JoinRequestStatusRejected {
		t.Errorf("expected REJECTED, got %s", out.Request.Status())
	}
}

func TestRejectRequest_RequestNotFoundReturns404(t *testing.T) {
	uc := newRejectUC(leadMemberRepo("g1", "lead-id"), &fakeJoinRequestRepo{})

	_, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "nonexistent",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND, got %v", err)
	}
}

func TestRejectRequest_AlreadyProcessedReturns400(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	_ = req.Approve()
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectUC(leadMemberRepo("g1", "lead-id"), reqRepo)

	_, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestAlreadyProcessed {
		t.Fatalf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestRejectRequest_SuccessUpdatesStatus(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newRejectUC(leadMemberRepo("g1", "lead-id"), reqRepo)

	out, err := uc.Execute(context.Background(), RejectRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request.Status() != domainGroup.JoinRequestStatusRejected {
		t.Errorf("expected REJECTED, got %s", out.Request.Status())
	}
	if len(reqRepo.savedRequests) == 0 {
		t.Error("expected request to be saved")
	}
}
