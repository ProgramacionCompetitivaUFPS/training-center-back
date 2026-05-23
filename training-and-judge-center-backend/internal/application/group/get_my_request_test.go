package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGetMyRequestUseCase(reqRepo *mockJoinRequestRepository) *GetMyRequestUseCase {
	return NewGetMyRequestUseCase(reqRepo)
}

func TestGetMyRequest_NoRequestReturns404(t *testing.T) {
	uc := NewGetMyRequestUseCase(&mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), GetMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND, got %v", err)
	}
}

func TestGetMyRequest_ReturnsOwnRequest(t *testing.T) {
	req := mustJoinRequest(t, "r1", "g1", "u1")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := NewGetMyRequestUseCase(reqRepo)

	out, err := uc.Execute(context.Background(), GetMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request.ID != "r1" {
		t.Errorf("expected request ID=r1, got %s", out.Request.ID)
	}
}

func TestGetMyRequest_OtherUsersRequestNotReturned(t *testing.T) {
	req := mustJoinRequest(t, "r1", "g1", "other-user")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := NewGetMyRequestUseCase(reqRepo)

	_, err := uc.Execute(context.Background(), GetMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND for another user's request, got %v", err)
	}
}
