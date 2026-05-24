package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)


func TestCancelMyRequest_NoRequestReturns404(t *testing.T) {
	uc := NewCancelMyRequestUseCase(&mockJoinRequestRepository{})

	err := uc.Execute(context.Background(), CancelMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND, got %v", err)
	}
}

func TestCancelMyRequest_AlreadyProcessedReturns400(t *testing.T) {
	req := mustJoinRequest(t, "r1", "g1", "u1")
	_ = req.Approve()
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := NewCancelMyRequestUseCase(reqRepo)

	err := uc.Execute(context.Background(), CancelMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestAlreadyProcessed {
		t.Fatalf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestCancelMyRequest_DeletesPendingRequest(t *testing.T) {
	req := mustJoinRequest(t, "r1", "g1", "u1")
	reqRepo := &mockJoinRequestRepository{requests: []*domainGroup.JoinRequest{req}}
	uc := NewCancelMyRequestUseCase(reqRepo)

	err := uc.Execute(context.Background(), CancelMyRequestInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reqRepo.deletedIDs) != 1 || reqRepo.deletedIDs[0] != "r1" {
		t.Errorf("expected request r1 to be deleted, deletedIDs=%v", reqRepo.deletedIDs)
	}
}
