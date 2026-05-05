package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newApproveUC(memberRepo *fakeMemberRepo, reqRepo *fakeJoinRequestRepo) *ApproveRequestUseCase {
	return NewApproveRequestUseCase(memberRepo, reqRepo, &fakeTxManager{})
}

func leadMemberRepo(groupID, userID string) *fakeMemberRepo {
	uid := shared.RestoreUserID(userID)
	lead, _ := domainGroup.NewGroupMember("m-lead", groupID, uid, domainGroup.MemberRoleLead, time.Now())
	return &fakeMemberRepo{memberships: map[string]*domainGroup.GroupMember{keyOf(groupID, uid): lead}}
}

func pendingRequest(id, groupID, requesterID string) *domainGroup.JoinRequest {
	req, _ := domainGroup.NewJoinRequest(id, groupID, shared.RestoreUserID(requesterID), nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return req
}

func TestApproveRequest_NonLeadReturns403(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newApproveUC(&fakeMemberRepo{}, reqRepo)

	_, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("not-lead", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestApproveRequest_AdminCanApproveWithoutMembership(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newApproveUC(&fakeMemberRepo{}, reqRepo)

	out, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("admin-id", shared.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("admin should be able to approve, got: %v", err)
	}
	if out.Request.Status() != domainGroup.JoinRequestStatusApproved {
		t.Errorf("expected APPROVED, got %s", out.Request.Status())
	}
}

func TestApproveRequest_RequestNotFoundReturns404(t *testing.T) {
	uc := newApproveUC(leadMemberRepo("g1", "lead-id"), &fakeJoinRequestRepo{})

	_, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "nonexistent",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND, got %v", err)
	}
}

func TestApproveRequest_WrongGroupIDReturns404(t *testing.T) {
	req := pendingRequest("r1", "other-group", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newApproveUC(leadMemberRepo("g1", "lead-id"), reqRepo)

	_, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestNotFound {
		t.Fatalf("expected REQUEST_NOT_FOUND, got %v", err)
	}
}

func TestApproveRequest_AlreadyProcessedReturns400(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	_ = req.Reject()
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	uc := newApproveUC(leadMemberRepo("g1", "lead-id"), reqRepo)

	_, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeRequestAlreadyProcessed {
		t.Fatalf("expected REQUEST_ALREADY_PROCESSED, got %v", err)
	}
}

func TestApproveRequest_SuccessCreatesMembership(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}
	memberRepo := leadMemberRepo("g1", "lead-id")
	uc := newApproveUC(memberRepo, reqRepo)

	out, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Request.Status() != domainGroup.JoinRequestStatusApproved {
		t.Errorf("expected APPROVED status, got %s", out.Request.Status())
	}
	if memberRepo.savedMember == nil {
		t.Error("expected a GroupMember to be saved, but none was")
	}
	if len(reqRepo.savedRequests) != 1 {
		t.Errorf("expected request to be saved once, got %d", len(reqRepo.savedRequests))
	}
}

func TestApproveRequest_AlreadyMemberReturns409(t *testing.T) {
	req := pendingRequest("r1", "g1", "requester-id")
	reqRepo := &fakeJoinRequestRepo{requests: []*domainGroup.JoinRequest{req}}

	requesterUID := shared.RestoreUserID("requester-id")
	existingMember, _ := domainGroup.NewGroupMember("m-existing", "g1", requesterUID, domainGroup.MemberRoleMember, time.Now())

	memberRepo := leadMemberRepo("g1", "lead-id")
	memberRepo.memberships[keyOf("g1", requesterUID)] = existingMember

	uc := newApproveUC(memberRepo, reqRepo)

	_, err := uc.Execute(context.Background(), ApproveRequestInput{
		GroupID:     "g1",
		RequestID:   "r1",
		CurrentUser: currentUser("lead-id", shared.RoleContestant),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
	if len(reqRepo.savedRequests) != 0 {
		t.Errorf("request should not have been saved on conflict, got %d saved", len(reqRepo.savedRequests))
	}
}
