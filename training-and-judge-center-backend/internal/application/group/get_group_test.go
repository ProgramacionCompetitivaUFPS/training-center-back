package group

import (
	"context"
	"errors"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func notVisibleGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Private Club", domainGroup.VisibilityNotVisible, domainGroup.JoinPolicyInvite)
}

func visibleGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Open Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
}

func TestGetGroup_EmptyGroupIDReturnsValidationError(t *testing.T) {
	uc := NewGetGroupUseCase(&mockGroupRepository{}, &mockMemberRepository{}, &mockUserProvider{})

	_, err := uc.Execute(context.Background(), GetGroupInput{
		GroupID:     "",
		CurrentUser: asContestant("u1"),
	})
	if err == nil {
		t.Fatal("expected validation error for empty groupId")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestGetGroup_UserProviderErrorReturnsInternal(t *testing.T) {
	g := visibleGroup(t)
	uidLead := shared.RestoreUserID("lead1")
	lead, _ := domainGroup.NewGroupMember("m1", "g1", uidLead, domainGroup.MemberRoleLead, testNow)

	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := &mockMemberRepository{
		memberCounts: map[string]int{"g1": 1},
		leadCounts:   map[string]int{"g1": 1},
		leads:        map[string][]*domainGroup.GroupMember{"g1": {lead}},
	}
	uc := NewGetGroupUseCase(repo, memberRepo, &mockUserProvider{err: errors.New("db down")})

	_, err := uc.Execute(context.Background(), GetGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("u1"),
	})
	if err == nil {
		t.Fatal("expected error when user provider fails")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestGetGroup_NotVisibleHiddenFromStranger(t *testing.T) {
	g := notVisibleGroup(t)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := &mockMemberRepository{}
	uc := NewGetGroupUseCase(repo, memberRepo, &mockUserProvider{})

	_, err := uc.Execute(context.Background(), GetGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("stranger"),
	})
	if err == nil {
		t.Fatal("expected 404 for non-member on NOT_VISIBLE group")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeGroupNotFound {
		t.Fatalf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestGetGroup_NotVisibleReturnedToMember(t *testing.T) {
	g := notVisibleGroup(t)
	uid := shared.RestoreUserID("member")
	gm, _ := domainGroup.NewGroupMember("m1", "g1", uid, domainGroup.MemberRoleMember, testNow)

	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := &mockMemberRepository{
		memberships:  map[string]*domainGroup.GroupMember{keyOf("g1", uid): gm},
		memberCounts: map[string]int{"g1": 5},
		leadCounts:   map[string]int{"g1": 1},
	}
	uc := NewGetGroupUseCase(repo, memberRepo, &mockUserProvider{})

	out, err := uc.Execute(context.Background(), GetGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("member"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Membership.Role == nil {
		t.Error("expected membership role to be set")
	}
	if out.Statistics.MemberCount != 5 || out.Statistics.LeadCount != 1 {
		t.Errorf("bad stats: %+v", out.Statistics)
	}
}

func TestGetGroup_AdminSeesNotVisible(t *testing.T) {
	g := notVisibleGroup(t)
	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := &mockMemberRepository{memberCounts: map[string]int{"g1": 0}, leadCounts: map[string]int{"g1": 0}}
	uc := NewGetGroupUseCase(repo, memberRepo, &mockUserProvider{})

	_, err := uc.Execute(context.Background(), GetGroupInput{
		GroupID:     "g1",
		CurrentUser: asAdmin("admin"),
	})
	if err != nil {
		t.Fatalf("admin should see NOT_VISIBLE, got %v", err)
	}
}

func TestGetGroup_LeadsPopulated(t *testing.T) {
	g := visibleGroup(t)
	uidLead := shared.RestoreUserID("lead1")
	lead, _ := domainGroup.NewGroupMember("m1", "g1", uidLead, domainGroup.MemberRoleLead, testNow)

	repo := &mockGroupRepository{groups: []*domainGroup.Group{g}}
	memberRepo := &mockMemberRepository{
		memberCounts: map[string]int{"g1": 1},
		leadCounts:   map[string]int{"g1": 1},
		leads:        map[string][]*domainGroup.GroupMember{"g1": {lead}},
	}
	userProvider := &mockUserProvider{
		displays: map[string]*UserDisplay{"lead1": {Nickname: "johnny", Name: "John Smith"}},
	}
	uc := NewGetGroupUseCase(repo, memberRepo, userProvider)

	out, err := uc.Execute(context.Background(), GetGroupInput{
		GroupID:     "g1",
		CurrentUser: asContestant("anyone"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Leads) != 1 || out.Leads[0].Nickname != "johnny" {
		t.Errorf("bad leads: %+v", out.Leads)
	}
}
