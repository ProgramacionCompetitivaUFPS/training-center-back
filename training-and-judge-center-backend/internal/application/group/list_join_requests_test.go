package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListRequestsUC(memberRepo *fakeMemberRepo, reqRepo *fakeJoinRequestRepo) *ListJoinRequestsUseCase {
	return NewListJoinRequestsUseCase(memberRepo, reqRepo, &fakeUserProvider{})
}

func TestListJoinRequests_NonLeadReturns403(t *testing.T) {
	uc := newListRequestsUC(&fakeMemberRepo{}, &fakeJoinRequestRepo{})

	_, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		CurrentUser: currentUser("nobody", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestListJoinRequests_AdminCanList(t *testing.T) {
	req := pendingRequest("r1", "g1", "u1")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newListRequestsUC(&fakeMemberRepo{}, reqRepo)

	out, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		CurrentUser: currentUser("admin", shared.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(out.Requests))
	}
}

func TestListJoinRequests_DefaultsToStatusPending(t *testing.T) {
	pending := pendingRequest("r1", "g1", "u1")
	rejected := pendingRequest("r2", "g1", "u2")
	_ = rejected.Reject()
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{pending, rejected}}
	uc := newListRequestsUC(leadMemberRepo("g1", "lead-id"), reqRepo)

	out, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Requests) != 1 {
		t.Errorf("expected 1 pending request, got %d", len(out.Requests))
	}
}

func TestListJoinRequests_InvalidStatusReturnsValidationError(t *testing.T) {
	uc := newListRequestsUC(leadMemberRepo("g1", "lead-id"), &fakeJoinRequestRepo{})

	_, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		Status:      "INVALID",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestListJoinRequests_LimitExceeded100Returns400(t *testing.T) {
	uc := newListRequestsUC(leadMemberRepo("g1", "lead-id"), &fakeJoinRequestRepo{})

	_, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		Limit:       101,
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}
