package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func makeAddMemberUseCase() (*AddMemberUseCase, *fakeRepo, *fakeMemberRepo, *fakeNicknameResolver) {
	repo := &fakeRepo{}
	memberRepo := &fakeMemberRepo{memberships: map[string]*domainGroup.GroupMember{}}
	resolver := &fakeNicknameResolver{users: map[string]*UserInfo{}}
	uc := NewAddMemberUseCase(repo, memberRepo, resolver, &fakeTxManager{})
	return uc, repo, memberRepo, resolver
}

func TestAddMember_NicknameNotFound(t *testing.T) {
	uc, repo, _, _ := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "ghost", Role: "MEMBER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	assertErrCode(t, err, domainGroup.ErrCodeNicknameNotFound)
}

func TestAddMember_GroupNotFound(t *testing.T) {
	uc, _, _, resolver := makeAddMemberUseCase()
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "nope", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	assertErrCode(t, err, domainGroup.ErrCodeGroupNotFound)
}

func TestAddMember_NonLeadForbidden(t *testing.T) {
	uc, repo, _, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("u-member", shared.RoleContestant),
	})
	assertErrCode(t, err, domainGroup.ErrCodeInsufficientPermissions)
}

func TestAddMember_ContestantCannotBePromotedToLead(t *testing.T) {
	uc, repo, memberRepo, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	lead := domainGroup.RestoreGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, time.Now(), nil, domainGroup.JoinMethodDirectAdd, nil)
	memberRepo.memberships[keyOf("g1", shared.RestoreUserID("lead1"))] = lead
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "LEAD",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	assertErrCode(t, err, domainGroup.ErrCodeInvalidLeadAssignment)
}

func TestAddMember_AlreadyMember(t *testing.T) {
	uc, repo, memberRepo, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	lead := domainGroup.RestoreGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, time.Now(), nil, domainGroup.JoinMethodDirectAdd, nil)
	existing := domainGroup.RestoreGroupMember("m2", "g1", shared.RestoreUserID("u2"), domainGroup.MemberRoleMember, time.Now(), nil, domainGroup.JoinMethodDirectAdd, nil)
	memberRepo.memberships[keyOf("g1", shared.RestoreUserID("lead1"))] = lead
	memberRepo.memberships[keyOf("g1", shared.RestoreUserID("u2"))] = existing
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	assertErrCode(t, err, domainGroup.ErrCodeAlreadyMember)
}

func TestAddMember_Success(t *testing.T) {
	uc, repo, memberRepo, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	lead := domainGroup.RestoreGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, time.Now(), nil, domainGroup.JoinMethodDirectAdd, nil)
	memberRepo.memberships[keyOf("g1", shared.RestoreUserID("lead1"))] = lead
	out, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Member.Role() != domainGroup.MemberRoleMember {
		t.Errorf("expected MEMBER role")
	}
	if out.Nickname != "alice" {
		t.Errorf("expected nickname=alice, got %q", out.Nickname)
	}
	if out.Member.AddedBy() == nil || out.Member.AddedBy().Value() != "lead1" {
		t.Errorf("expected AddedBy=lead1, got %v", out.Member.AddedBy())
	}
}

func TestAddMember_AdminBypassesLeadCheck(t *testing.T) {
	uc, repo, _, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	// admin is NOT a member of the group — should still succeed
	out, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("admin1", shared.RoleAdmin),
	})
	if err != nil {
		t.Fatalf("admin should be able to add members without being a lead: %v", err)
	}
	if out.Member.Role() != domainGroup.MemberRoleMember {
		t.Errorf("expected MEMBER role")
	}
}

func TestAddMember_InvalidRole(t *testing.T) {
	uc, repo, memberRepo, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleContestant}
	lead := domainGroup.RestoreGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, time.Now(), nil, domainGroup.JoinMethodDirectAdd, nil)
	memberRepo.memberships[keyOf("g1", shared.RestoreUserID("lead1"))] = lead
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "SUPERUSER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestAddMember_ResolverError(t *testing.T) {
	repo := &fakeRepo{groups: []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}}
	memberRepo := &fakeMemberRepo{memberships: map[string]*domainGroup.GroupMember{}}
	resolver := &fakeNicknameResolver{users: map[string]*UserInfo{}, err: apperror.NewInternal()}
	uc := NewAddMemberUseCase(repo, memberRepo, resolver, &fakeTxManager{})
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	if err == nil {
		t.Fatal("expected error to propagate from resolver")
	}
}

func TestAddMember_CoachCanBePromotedToLead(t *testing.T) {
	uc, repo, memberRepo, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["coach1"] = &UserInfo{ID: shared.RestoreUserID("u2"), Role: shared.RoleCoach}
	lead := domainGroup.RestoreGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, time.Now(), nil, domainGroup.JoinMethodDirectAdd, nil)
	memberRepo.memberships[keyOf("g1", shared.RestoreUserID("lead1"))] = lead
	out, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "coach1", Role: "LEAD",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	if err != nil {
		t.Fatalf("coach should be assignable as lead: %v", err)
	}
	if out.Member.Role() != domainGroup.MemberRoleLead {
		t.Errorf("expected LEAD role")
	}
}
