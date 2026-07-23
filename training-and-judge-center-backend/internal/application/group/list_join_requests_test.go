package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newListJoinRequestsUseCase(memberRepo *mockMemberRepository, reqRepo *mockJoinRequestRepository) *ListJoinRequestsUseCase {
	return NewListJoinRequestsUseCase(memberRepo, reqRepo, &mockUserProvider{})
}

func TestListJoinRequests_NonLeadReturns403(t *testing.T) {
	uc := newListJoinRequestsUseCase(&mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		CurrentUser: asContestant("nobody"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestListJoinRequests_AdminCanList(t *testing.T) {
	req := pendingRequest("r1", "g1", "u1")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := newListJoinRequestsUseCase(&mockMemberRepository{}, reqRepo)

	out, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asAdmin("admin"),
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
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{pending, rejected}}
	uc := newListJoinRequestsUseCase(leadMemberRepo("g1", "lead-id"), reqRepo)

	out, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		Page:        1,
		Limit:       20,
		CurrentUser: asContestant("lead-id"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Requests) != 1 {
		t.Errorf("expected 1 pending request, got %d", len(out.Requests))
	}
}

func TestListJoinRequests_InvalidStatusReturnsValidationError(t *testing.T) {
	uc := newListJoinRequestsUseCase(leadMemberRepo("g1", "lead-id"), &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		Status:      "INVALID",
		CurrentUser: asContestant("lead-id"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestListJoinRequests_LimitExceeded100Returns400(t *testing.T) {
	uc := newListJoinRequestsUseCase(leadMemberRepo("g1", "lead-id"), &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), ListJoinRequestsInput{
		GroupID:     "g1",
		Limit:       101,
		CurrentUser: asContestant("lead-id"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}
