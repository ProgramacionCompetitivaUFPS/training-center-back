package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func strPtr(s string) *string { return &s }

func mustGroupNoT(id, name string, vis domainGroup.Visibility, jp domainGroup.JoinPolicy) *domainGroup.Group {
	gn, _ := domainGroup.NewGroupName(name)
	g, _ := domainGroup.NewGroup(id, gn, nil, vis, jp, shared.RestoreUserID("creator-id"), testNow)
	return g
}

func updateUC(
	repo *mockGroupRepository,
	memberRepo *mockMemberRepository,
	reqRepo *mockJoinRequestRepository,
) *UpdateGroupUseCase {
	return NewUpdateGroupUseCase(repo, memberRepo, reqRepo, &mockTransactionManager{})
}

func TestUpdateGroup_GroupNotFoundReturns404(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := updateUC(repo, &mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "missing",
		Name:        strPtr("New Name"),
		CurrentUser: asAdmin("admin-1"),
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
		t.Fatalf("expected KindNotFound, got %v", err)
	}
}

func TestUpdateGroup_GlobalGroupReturns400(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-default", domainGroup.RestoreGroupName("Global"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen,
		true, shared.RestoreUserID("sys"), testNow, testNow,
	)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := updateUC(repo, &mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g-default",
		Name:        strPtr("New"),
		CurrentUser: asAdmin("admin-1"),
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainGroup.ErrCodeCannotModifyGlobalGroup {
		t.Fatalf("expected CANNOT_MODIFY_GLOBAL_GROUP, got %v", err)
	}
}

func TestUpdateGroup_NonLeadReturns403(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := updateUC(repo, &mockMemberRepository{}, &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		Name:        strPtr("Beta"),
		CurrentUser: asContestant("user-1"),
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Fatalf("expected KindForbidden, got %v", err)
	}
}

func TestUpdateGroup_NoFieldsReturns400(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		CurrentUser: asAdmin("admin-1"),
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Fatalf("expected KindValidation, got %v", err)
	}
}

func TestUpdateGroup_DuplicateNameReturns409(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	repo := &mockGroupRepository{
		groups:         []*domainGroup.Group{g},
		existsByNameFn: func(_ domainGroup.GroupName) (bool, error) { return true, nil },
	}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		Name:        strPtr("Beta Team"),
		CurrentUser: asAdmin("admin-1"),
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainGroup.ErrCodeNameAlreadyExists {
		t.Fatalf("expected NAME_ALREADY_EXISTS, got %v", err)
	}
}

func TestUpdateGroup_InvalidPolicyCombinationReturns400(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityNotVisible, domainGroup.JoinPolicyInvite)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), &mockJoinRequestRepository{})

	_, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		JoinPolicy:  strPtr("OPEN"),
		CurrentUser: asAdmin("admin-1"),
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainGroup.ErrCodeInvalidPolicyCombination {
		t.Fatalf("expected INVALID_POLICY_COMBINATION, got %v", err)
	}
}

func TestUpdateGroup_UpdateNameSucceeds(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), &mockJoinRequestRepository{})

	out, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		Name:        strPtr("Beta Team"),
		CurrentUser: asAdmin("admin-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Beta Team" {
		t.Errorf("expected Name=Beta Team, got %s", out.Name)
	}
}

func TestUpdateGroup_RequestToOpenAutoApprovesRequests(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	reqRepo := &mockJoinRequestRepository{
		requests: []*domainGroup.JoinRequest{
			pendingRequest("r1", "g1", "user-a"),
			pendingRequest("r2", "g1", "user-b"),
		},
	}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), reqRepo)

	out, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		JoinPolicy:  strPtr("OPEN"),
		CurrentUser: asAdmin("admin-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RequestsAutoApproved != 2 {
		t.Errorf("expected 2 auto-approved, got %d", out.RequestsAutoApproved)
	}
	if out.JoinPolicy != "OPEN" {
		t.Errorf("expected JoinPolicy=OPEN, got %s", out.JoinPolicy)
	}
}

func TestUpdateGroup_RequestToInviteAutoRejectsRequests(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	reqRepo := &mockJoinRequestRepository{
		requests: []*domainGroup.JoinRequest{
			pendingRequest("r1", "g1", "user-a"),
		},
	}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), reqRepo)

	out, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		JoinPolicy:  strPtr("INVITE"),
		CurrentUser: asAdmin("admin-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RequestsAutoRejected != 1 {
		t.Errorf("expected 1 auto-rejected, got %d", out.RequestsAutoRejected)
	}
}

func TestUpdateGroup_NotVisibleForcesInviteAndRejectsRequests(t *testing.T) {
	g := mustGroupNoT("g1", "Alpha", domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	reqRepo := &mockJoinRequestRepository{
		requests: []*domainGroup.JoinRequest{
			pendingRequest("r1", "g1", "user-a"),
			pendingRequest("r2", "g1", "user-b"),
		},
	}
	uc := updateUC(repo, leadMemberRepo("g1", "admin-1"), reqRepo)

	out, err := uc.Execute(context.Background(), UpdateGroupInput{
		GroupID:     "g1",
		Visibility:  strPtr("NOT_VISIBLE"),
		CurrentUser: asAdmin("admin-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Visibility != "NOT_VISIBLE" {
		t.Errorf("expected Visibility=NOT_VISIBLE, got %s", out.Visibility)
	}
	if out.JoinPolicy != "INVITE" {
		t.Errorf("expected JoinPolicy forced to INVITE, got %s", out.JoinPolicy)
	}
	if out.RequestsAutoRejected != 2 {
		t.Errorf("expected 2 auto-rejected, got %d", out.RequestsAutoRejected)
	}
}
