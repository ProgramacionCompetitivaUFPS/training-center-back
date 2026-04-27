package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
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
	resolver.users["alice"] = &UserInfo{ID: "u2", Role: shared.RoleContestant}
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "nope", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("lead1", shared.RoleCoach),
	})
	assertErrCode(t, err, domainGroup.ErrCodeGroupNotFound)
}

func TestAddMember_NonLeadForbidden(t *testing.T) {
	uc, repo, _, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: "u2", Role: shared.RoleContestant}
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID: "g1", Nickname: "alice", Role: "MEMBER",
		CurrentUser: currentUser("u-member", shared.RoleContestant),
	})
	assertErrCode(t, err, domainGroup.ErrCodeInsufficientPermissions)
}

func TestAddMember_ContestantCannotBePromotedToLead(t *testing.T) {
	uc, repo, memberRepo, resolver := makeAddMemberUseCase()
	repo.groups = []*domainGroup.Group{mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)}
	resolver.users["alice"] = &UserInfo{ID: "u2", Role: shared.RoleContestant}
	lead, _ := domainGroup.NewGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, nil, domainGroup.JoinMethodDirectAdd, nil)
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
	resolver.users["alice"] = &UserInfo{ID: "u2", Role: shared.RoleContestant}
	lead, _ := domainGroup.NewGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, nil, domainGroup.JoinMethodDirectAdd, nil)
	existing, _ := domainGroup.NewGroupMember("m2", "g1", shared.RestoreUserID("u2"), domainGroup.MemberRoleMember, nil, domainGroup.JoinMethodDirectAdd, nil)
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
	resolver.users["alice"] = &UserInfo{ID: "u2", Role: shared.RoleContestant}
	lead, _ := domainGroup.NewGroupMember("m1", "g1", shared.RestoreUserID("lead1"), domainGroup.MemberRoleLead, nil, domainGroup.JoinMethodDirectAdd, nil)
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
}
