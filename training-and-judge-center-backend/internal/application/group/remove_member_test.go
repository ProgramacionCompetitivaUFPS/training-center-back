package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newRemoveMemberUC(groupRepo *mockGroupRepository, memberRepo *mockMemberRepository, resolver *mockNicknameResolver) *RemoveMemberUseCase {
	return NewRemoveMemberUseCase(groupRepo, memberRepo, resolver, &mockContestRegistrationCleaner{}, &mockTransactionManager{})
}

func TestRemoveMember_NonLeadReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "CONTESTANT"}}
	uc := newRemoveMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, &mockMemberRepository{}, resolver)

	err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g1",
		Nickname:    "target",
		CurrentUser: asContestant("not-lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestRemoveMember_GlobalGroupReturns400(t *testing.T) {
	g := domainGroup.RestoreGroup("g-global", domainGroup.RestoreGroupName("Global"),
		nil, domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen, true,
		mustUID("creator"), testNow, testNow)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target"}}
	uc := newRemoveMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, leadMemberRepo("g-global", "lead"), resolver)

	err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g-global",
		Nickname:    "target",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeCannotRemoveFromGlobalGroup {
		t.Fatalf("expected CANNOT_REMOVE_FROM_GLOBAL_GROUP, got %v", err)
	}
}

func TestRemoveMember_NicknameNotFoundReturns404(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newRemoveMemberUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "lead"),
		&mockNicknameResolver{},
	)
	err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g1",
		Nickname:    "ghost",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != "NICKNAME_NOT_FOUND" {
		t.Fatalf("expected NICKNAME_NOT_FOUND, got %v", err)
	}
}

func TestRemoveMember_NotAMemberReturns404(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target"}}
	uc := newRemoveMemberUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "lead"),
		resolver,
	)
	err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g1",
		Nickname:    "target",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeNotAMember {
		t.Fatalf("expected NOT_A_MEMBER, got %v", err)
	}
}

func TestRemoveMember_LastLeadReturns400(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "lead", Nickname: "lead"}}
	memberRepo := leadMemberRepo("g1", "lead")
	memberRepo.leadCounts = map[string]int{"g1": 1}
	uc := newRemoveMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, resolver)

	err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g1",
		Nickname:    "lead",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeCannotRemoveLastLead {
		t.Fatalf("expected CANNOT_REMOVE_LAST_LEAD, got %v", err)
	}
}

func TestRemoveMember_SuccessUnregistersFromScheduledContests(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "member2"}}
	memberRepo := leadMemberRepo("g1", "lead")
	member2, _ := domainGroup.NewGroupMember("m2", "g1", mustUID("u2"), domainGroup.MemberRoleMember, domainGroup.JoinMethodOpenJoin, nil, testNow)
	memberRepo.memberships[keyOf("g1", mustUID("u2"))] = member2

	cleaner := &mockContestRegistrationCleaner{}
	uc := NewRemoveMemberUseCase(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		memberRepo, resolver, cleaner, &mockTransactionManager{},
	)
	if err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g1",
		Nickname:    "member2",
		CurrentUser: asCoach("lead"),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cleaner.called {
		t.Error("expected contestCleaner.DeleteScheduledByGroupAndUser to be called")
	}
}

func TestRemoveMember_SuccessRegularMember(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "member2"}}
	memberRepo := leadMemberRepo("g1", "lead")
	member2, _ := domainGroup.NewGroupMember("m2", "g1", mustUID("u2"), domainGroup.MemberRoleMember, domainGroup.JoinMethodOpenJoin, nil, testNow)
	memberRepo.memberships[keyOf("g1", mustUID("u2"))] = member2

	uc := newRemoveMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, resolver)
	err := uc.Execute(context.Background(), RemoveMemberInput{
		GroupID:     "g1",
		Nickname:    "member2",
		CurrentUser: asCoach("lead"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
