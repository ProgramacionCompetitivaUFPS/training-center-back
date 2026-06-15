package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newDeleteGroupUseCase(
	groupRepo *mockGroupRepository,
	memberRepo *mockMemberRepository,
	joinRequestRepo *mockJoinRequestRepository,
	provider *mockGroupDeletionProvider,
) *DeleteGroupUseCase {
	return NewDeleteGroupUseCase(
		groupRepo,
		memberRepo,
		joinRequestRepo,
		provider,
		&mockGroupStandingsInvalidator{},
		&mockTransactionManager{},
	)
}

func openGroupForDelete(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "My Group", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
}

func defaultCounts() DeletionCounts {
	return DeletionCounts{
		ContestIDs:       []string{"c1", "c2"},
		ContestsCount:    2,
		MaterialsCount:   3,
		SubmissionsCount: 10,
		MembersCount:     5,
	}
}

func TestDeleteGroup_LeadDeletesSuccessfully(t *testing.T) {
	g := openGroupForDelete(t)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := leadMemberRepo("g1", "lead-1")
	provider := &mockGroupDeletionProvider{counts: defaultCounts()}

	uc := newDeleteGroupUseCase(groupRepo, memberRepo, &mockJoinRequestRepository{}, provider)

	out, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "g1",
		ConfirmationName: "My Group",
		CurrentUser:      asCoach("lead-1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.GroupID != "g1" {
		t.Errorf("GroupID = %q, want %q", out.GroupID, "g1")
	}
	if out.GroupName != "My Group" {
		t.Errorf("GroupName = %q, want %q", out.GroupName, "My Group")
	}
	if out.ContestsDeleted != 2 {
		t.Errorf("ContestsDeleted = %d, want 2", out.ContestsDeleted)
	}
	if out.MaterialsDeleted != 3 {
		t.Errorf("MaterialsDeleted = %d, want 3", out.MaterialsDeleted)
	}
	if out.SubmissionsOrphaned != 10 {
		t.Errorf("SubmissionsOrphaned = %d, want 10", out.SubmissionsOrphaned)
	}
	if out.MembersRemoved != 5 {
		t.Errorf("MembersRemoved = %d, want 5", out.MembersRemoved)
	}
	if out.StandingsDeleted != 2 {
		t.Errorf("StandingsDeleted = %d, want 2", out.StandingsDeleted)
	}
}

func TestDeleteGroup_AdminDeletesGroupTheyDontBelongTo(t *testing.T) {
	g := openGroupForDelete(t)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	provider := &mockGroupDeletionProvider{counts: defaultCounts()}

	uc := newDeleteGroupUseCase(groupRepo, &mockMemberRepository{}, &mockJoinRequestRepository{}, provider)

	out, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "g1",
		ConfirmationName: "My Group",
		CurrentUser:      asAdmin("admin-1"),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.GroupID != "g1" {
		t.Errorf("GroupID = %q, want %q", out.GroupID, "g1")
	}
}

func TestDeleteGroup_GroupNotFound(t *testing.T) {
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{}}
	uc := newDeleteGroupUseCase(groupRepo, &mockMemberRepository{}, &mockJoinRequestRepository{}, &mockGroupDeletionProvider{})

	_, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "nonexistent",
		ConfirmationName: "anything",
		CurrentUser:      asAdmin("admin-1"),
	})

	if err == nil {
		t.Fatal("expected error for missing group")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindNotFound {
		t.Errorf("expected NOT_FOUND, got %v", err)
	}
}

func TestDeleteGroup_CannotDeleteGlobalGroup(t *testing.T) {
	gn, _ := domainGroup.NewGroupName("Global Group")
	global := domainGroup.RestoreGroup(
		"global", gn, nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen,
		true, shared.RestoreUserID("system"), time.Now(), time.Now(),
	)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{global}}
	uc := newDeleteGroupUseCase(groupRepo, &mockMemberRepository{}, &mockJoinRequestRepository{}, &mockGroupDeletionProvider{})

	_, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "global",
		ConfirmationName: "Global Group",
		CurrentUser:      asAdmin("admin-1"),
	})

	if err == nil {
		t.Fatal("expected error when deleting global group")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected FORBIDDEN, got %v", err)
	}
	if ae.Code != domainGroup.ErrCodeCannotDeleteGlobalGroup {
		t.Errorf("expected code %s, got %s", domainGroup.ErrCodeCannotDeleteGlobalGroup, ae.Code)
	}
}

func TestDeleteGroup_NonLeadMemberForbidden(t *testing.T) {
	g := openGroupForDelete(t)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uid := shared.RestoreUserID("member-1")
	member, _ := domainGroup.NewGroupMember("m1", "g1", uid, domainGroup.MemberRoleMember, domainGroup.JoinMethodOpenJoin, nil, testNow)
	memberRepo := &mockMemberRepository{
		memberships: map[string]*domainGroup.GroupMember{keyOf("g1", uid): member},
	}
	uc := newDeleteGroupUseCase(groupRepo, memberRepo, &mockJoinRequestRepository{}, &mockGroupDeletionProvider{})

	_, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "g1",
		ConfirmationName: "My Group",
		CurrentUser:      asCoach("member-1"),
	})

	if err == nil {
		t.Fatal("expected forbidden error for non-lead member")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected FORBIDDEN, got %v", err)
	}
}

func TestDeleteGroup_NonMemberForbidden(t *testing.T) {
	g := openGroupForDelete(t)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	uc := newDeleteGroupUseCase(groupRepo, &mockMemberRepository{}, &mockJoinRequestRepository{}, &mockGroupDeletionProvider{})

	_, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "g1",
		ConfirmationName: "My Group",
		CurrentUser:      asCoach("outsider-1"),
	})

	if err == nil {
		t.Fatal("expected forbidden error for non-member")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindForbidden {
		t.Errorf("expected FORBIDDEN, got %v", err)
	}
}

func TestDeleteGroup_ConfirmationEmpty(t *testing.T) {
	g := openGroupForDelete(t)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := leadMemberRepo("g1", "lead-1")
	uc := newDeleteGroupUseCase(groupRepo, memberRepo, &mockJoinRequestRepository{}, &mockGroupDeletionProvider{})

	_, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "g1",
		ConfirmationName: "",
		CurrentUser:      asCoach("lead-1"),
	})

	if err == nil {
		t.Fatal("expected validation error for empty confirmation")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindValidation {
		t.Errorf("expected VALIDATION, got %v", err)
	}
}

func TestDeleteGroup_ConfirmationMismatch(t *testing.T) {
	g := openGroupForDelete(t)
	groupRepo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := leadMemberRepo("g1", "lead-1")
	uc := newDeleteGroupUseCase(groupRepo, memberRepo, &mockJoinRequestRepository{}, &mockGroupDeletionProvider{})

	_, err := uc.Execute(context.Background(), DeleteGroupInput{
		GroupID:          "g1",
		ConfirmationName: "wrong name",
		CurrentUser:      asCoach("lead-1"),
	})

	if err == nil {
		t.Fatal("expected error for confirmation mismatch")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Kind != apperror.KindBadRequest {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
	if ae.Code != ErrCodeConfirmationMismatch {
		t.Errorf("expected code %s, got %s", ErrCodeConfirmationMismatch, ae.Code)
	}
}
