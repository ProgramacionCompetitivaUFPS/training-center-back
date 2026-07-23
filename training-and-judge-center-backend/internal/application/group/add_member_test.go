package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newAddMemberUC(groupRepo *mockGroupRepository, memberRepo *mockMemberRepository, resolver *mockNicknameResolver) *AddMemberUseCase {
	return NewAddMemberUseCase(groupRepo, memberRepo, resolver)
}

func TestAddMember_EmptyNicknameReturnsValidation(t *testing.T) {
	uc := newAddMemberUC(&mockGroupRepository{}, &mockMemberRepository{}, &mockNicknameResolver{})
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "",
		Role:        "MEMBER",
		CurrentUser: asAdmin("admin"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestAddMember_EmptyRoleReturnsValidation(t *testing.T) {
	uc := newAddMemberUC(&mockGroupRepository{}, &mockMemberRepository{}, &mockNicknameResolver{})
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "user1",
		Role:        "",
		CurrentUser: asAdmin("admin"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestAddMember_NonLeadReturns403(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "CONTESTANT"}}
	uc := newAddMemberUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		&mockMemberRepository{},
		resolver,
	)
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "target",
		Role:        "MEMBER",
		CurrentUser: asContestant("not-lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestAddMember_AdminCanAddWithoutBeingLead(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "CONTESTANT"}}
	memberRepo := &mockMemberRepository{}
	uc := newAddMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, resolver)

	out, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "target",
		Role:        "MEMBER",
		CurrentUser: asAdmin("admin"),
	})
	if err != nil {
		t.Fatalf("admin should add without being lead, got: %v", err)
	}
	if out.Role != "MEMBER" {
		t.Errorf("expected MEMBER, got %s", out.Role)
	}
	if out.JoinMethod != "DIRECT_ADD" {
		t.Errorf("expected DIRECT_ADD, got %s", out.JoinMethod)
	}
	if memberRepo.savedMember == nil {
		t.Error("expected member to be saved")
	}
}

func TestAddMember_NicknameNotFoundReturns404(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	uc := newAddMemberUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "lead"),
		&mockNicknameResolver{},
	)
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "ghost",
		Role:        "MEMBER",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != "NICKNAME_NOT_FOUND" {
		t.Fatalf("expected NICKNAME_NOT_FOUND, got %v", err)
	}
}

func TestAddMember_AlreadyMemberReturns409(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "target", SystemRole: "CONTESTANT"}}
	memberRepo := leadMemberRepo("g1", "lead")
	// pre-load target as existing member
	existing, _ := domainGroup.NewGroupMember("m2", "g1", mustUID("u2"), domainGroup.MemberRoleMember, domainGroup.JoinMethodOpenJoin, nil, testNow)
	memberRepo.memberships[keyOf("g1", mustUID("u2"))] = existing

	uc := newAddMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, resolver)
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "target",
		Role:        "MEMBER",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeAlreadyMember {
		t.Fatalf("expected ALREADY_MEMBER, got %v", err)
	}
}

func TestAddMember_ContestantAsLeadReturns400(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "contestant", SystemRole: "CONTESTANT"}}
	uc := newAddMemberUC(
		&mockGroupRepository{groups: []*domainGroup.Group{g}},
		leadMemberRepo("g1", "lead"),
		resolver,
	)
	_, err := uc.Execute(context.Background(), AddMemberInput{
		GroupID:     "g1",
		Nickname:    "contestant",
		Role:        "LEAD",
		CurrentUser: asCoach("lead"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvalidLeadAssignment {
		t.Fatalf("expected INVALID_LEAD_ASSIGNMENT, got %v", err)
	}
}

func TestAddMember_CoachCanBeAddedAsLead(t *testing.T) {
	g := mustGroup(t, "g1", "Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen)
	resolver := &mockNicknameResolver{user: &UserDisplay{ID: "u2", Nickname: "coach2", SystemRole: "COACH"}}
	memberRepo := leadMemberRepo("g1", "lead")
	uc := newAddMemberUC(&mockGroupRepository{groups: []*domainGroup.Group{g}}, memberRepo, resolver)

	out, err := uc.Execute(context.Background(), AddMemberInput{
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
