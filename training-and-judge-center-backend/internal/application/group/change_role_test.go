package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newChangeRoleUC(groupRepo *mockGroupRepository, memberRepo *mockMemberRepository, resolver *mockNicknameResolver) *ChangeRoleUseCase {
	return NewChangeRoleUseCase(groupRepo, memberRepo, resolver)
}

func memberRepoWithMember(groupID, leadID, targetID string, targetRole domainGroup.MemberRole) *mockMemberRepository {
	repo := leadMemberRepo(groupID, leadID)
	m, _ := domainGroup.NewGroupMember("m-target", groupID, mustUID(targetID), targetRole, domainGroup.JoinMethodOpenJoin, nil, testNow)
	repo.memberships[keyOf(groupID, mustUID(targetID))] = m
	return repo
}

func TestChangeRole_EmptyRoleReturnsValidation(t *testing.T) {
	uc := newChangeRoleUC(&mockGroupRepository{}, &mockMemberRepository{}, &mockNicknameResolver{})
	_, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "user",
		Role:        "",
		CurrentUser: asAdmin("admin"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestChangeRole_NonLeadReturns403(t *testing.T) {
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "COACH"}}
	uc := newChangeRoleUC(&mockGroupRepository{}, &mockMemberRepository{}, resolver)
	_, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "target",
		Role:        "LEAD",
		CurrentUser: asContestant("not-lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestChangeRole_ContestantAsLeadReturns400(t *testing.T) {
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "CONTESTANT"}}
	uc := newChangeRoleUC(&mockGroupRepository{}, leadMemberRepo("g1", "lead"), resolver)
	_, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "target",
		Role:        "LEAD",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvalidLeadAssignment {
		t.Fatalf("expected INVALID_LEAD_ASSIGNMENT, got %v", err)
	}
}

func TestChangeRole_NotAMemberReturns404(t *testing.T) {
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "COACH"}}
	uc := newChangeRoleUC(&mockGroupRepository{}, leadMemberRepo("g1", "lead"), resolver)
	_, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "target",
		Role:        "LEAD",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeNotAMember {
		t.Fatalf("expected NOT_A_MEMBER, got %v", err)
	}
}

func TestChangeRole_DemoteLastLeadReturns400(t *testing.T) {
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "lead", Nickname: "lead", SystemRole: "COACH"}}
	memberRepo := leadMemberRepo("g1", "lead")
	memberRepo.leadCounts = map[string]int{"g1": 1}
	uc := newChangeRoleUC(&mockGroupRepository{}, memberRepo, resolver)
	_, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "lead",
		Role:        "MEMBER",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeCannotRemoveLastLead {
		t.Fatalf("expected CANNOT_REMOVE_LAST_LEAD, got %v", err)
	}
}

func TestChangeRole_PromoteMemberToLead(t *testing.T) {
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "coach2", SystemRole: "COACH"}}
	memberRepo := memberRepoWithMember("g1", "lead", "u2", domainGroup.MemberRoleMember)
	uc := newChangeRoleUC(&mockGroupRepository{}, memberRepo, resolver)
	out, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "coach2",
		Role:        "LEAD",
		CurrentUser: asCoach("lead"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Role != "LEAD" {
		t.Errorf("expected LEAD, got %s", out.Role)
	}
}

func TestChangeRole_DemoteLeadToMember(t *testing.T) {
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "lead2", SystemRole: "COACH"}}
	memberRepo := memberRepoWithMember("g1", "lead", "u2", domainGroup.MemberRoleLead)
	memberRepo.leadCounts = map[string]int{"g1": 2}
	uc := newChangeRoleUC(&mockGroupRepository{}, memberRepo, resolver)
	out, err := uc.Execute(context.Background(), ChangeRoleInput{
		GroupID:     "g1",
		Nickname:    "lead2",
		Role:        "MEMBER",
		CurrentUser: asCoach("lead"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Role != "MEMBER" {
		t.Errorf("expected MEMBER, got %s", out.Role)
	}
}
