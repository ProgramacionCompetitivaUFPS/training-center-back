package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
)


func TestGetMyRequest_NoRequestReturnsNil(t *testing.T) {
	uc := NewGetMyRequestUseCase(&mockJoinRequestRepository{})

	out, err := uc.Execute(context.Background(), GetMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request != nil {
		t.Errorf("expected nil request, got %+v", out.Request)
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
	if out.Request == nil || out.Request.ID != "r1" {
		t.Errorf("expected request ID=r1, got %v", out.Request)
	}
}

func TestGetMyRequest_OtherUsersRequestNotReturned(t *testing.T) {
	req := mustJoinRequest(t, "r1", "g1", "other-user")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := NewGetMyRequestUseCase(reqRepo)

	out, err := uc.Execute(context.Background(), GetMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request != nil {
		t.Errorf("expected nil request for another user's request, got %+v", out.Request)
	}
}
