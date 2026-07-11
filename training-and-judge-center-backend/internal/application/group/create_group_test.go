package group

import (
	"context"
	"testing"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validCreateInput(role shared.Role) CreateGroupInput {
	cu := asContestant("u1")
	switch role {
	case shared.RoleAdmin:
		cu = asAdmin("u1")
	case shared.RoleCoach:
		cu = asCoach("u1")
	}
	return CreateGroupInput{
		Name:        "Algorithms Club",
		Description: nil,
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: cu,
	}
}

func newCreateGroupUseCase(repo *mockGroupRepository, memberRepo *mockMemberRepository, nicknameResolver *mockNicknameResolver) *CreateGroupUseCase {
	if memberRepo == nil {
		memberRepo = &mockMemberRepository{}
	}
	if nicknameResolver == nil {
		nicknameResolver = &mockNicknameResolver{}
	}
	return NewCreateGroupUseCase(repo, memberRepo, nicknameResolver, &mockTransactionManager{})
}

func TestCreateGroup_NonAdminNonCoachReturns403(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: asContestant("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeInsufficientPermissions {
		t.Fatalf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateGroup_ValidCoachCreatesGroup(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := newCreateGroupUseCase(repo, nil, nil)

	out, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Algorithms Club" {
		t.Errorf("Name = %q, want %q", out.Name, "Algorithms Club")
	}
	if out.JoinPolicy != "OPEN" {
		t.Errorf("JoinPolicy = %q, want OPEN", out.JoinPolicy)
	}
	if out.Visibility != "VISIBLE" {
		t.Errorf("Visibility = %q, want VISIBLE", out.Visibility)
	}
	if out.ID == "" {
		t.Error("expected non-empty group ID")
	}
}

func TestCreateGroup_ValidAdminCreatesGroup(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := newCreateGroupUseCase(repo, nil, nil)

	out, err := uc.Execute(context.Background(), validCreateInput(shared.RoleAdmin))
	if err != nil {
		t.Fatalf("admin should be able to create groups, got: %v", err)
	}
	if out.ID == "" {
		t.Error("expected non-empty group ID")
	}
}

func TestCreateGroup_EmptyNameReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "",
		JoinMode:    "OPEN",
		Visibility:  "VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "name" {
		t.Errorf("expected field error on 'name', got %+v", ae.Details)
	}
}

func TestCreateGroup_InvalidJoinModeReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "INVALID_MODE",
		Visibility:  "VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "joinPolicy" {
		t.Errorf("expected field error on 'joinPolicy', got %+v", ae.Details)
	}
}

func TestCreateGroup_InvalidVisibilityReturnsValidationError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "My Group",
		JoinMode:    "OPEN",
		Visibility:  "INVISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "visibility" {
		t.Errorf("expected field error on 'visibility', got %+v", ae.Details)
	}
}

func TestCreateGroup_MultipleInvalidFieldsAccumulated(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "",
		JoinMode:    "NOPE",
		Visibility:  "VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected VALIDATION_ERROR, got %v", err)
	}
	if len(ae.Details) != 2 {
		t.Errorf("expected 2 field errors (name + joinPolicy), got %d: %+v", len(ae.Details), ae.Details)
	}
}

func TestCreateGroup_DuplicateNameReturnsConflict(t *testing.T) {
	repo := &mockGroupRepository{
		existsByNameFn: func(_ domainGroup.GroupName) (bool, error) { return true, nil },
	}
	uc := newCreateGroupUseCase(repo, nil, nil)

	_, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeNameAlreadyExists {
		t.Fatalf("expected NAME_ALREADY_EXISTS, got %v", err)
	}
}

func TestCreateGroup_RepoSaveFailureReturnsInternal(t *testing.T) {
	repo := &mockGroupRepository{saveErr: apperror.NewInternal()}
	uc := newCreateGroupUseCase(repo, nil, nil)

	_, err := uc.Execute(context.Background(), validCreateInput(shared.RoleCoach))
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR on save failure, got %v", err)
	}
}

func TestCreateGroup_InvalidPolicyCombinationReturnsError(t *testing.T) {
	uc := newCreateGroupUseCase(&mockGroupRepository{}, nil, nil)

	_, err := uc.Execute(context.Background(), CreateGroupInput{
		Name:        "Secret Club",
		JoinMode:    "OPEN",
		Visibility:  "NOT_VISIBLE",
		CurrentUser: asCoach("u1"),
	})
	ae, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if ae.Code != apperror.ErrCodeValidationError {
		t.Errorf("Code = %q, want VALIDATION_ERROR", ae.Code)
	}
	if len(ae.Details) == 0 || ae.Details[0].Field != "joinPolicy" {
		t.Errorf("expected joinPolicy field error, got %+v", ae.Details)
	}
}

func TestCreateGroup_MemberNicknamesAddedSuccessfully(t *testing.T) {
	repo := &mockGroupRepository{}
	memberRepo := &mockMemberRepository{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "member-1", Nickname: "alice", Name: "Alice", SystemRole: shared.RoleContestant.String()},
		"bob":   {ID: "member-2", Nickname: "bob", Name: "Bob", SystemRole: shared.RoleContestant.String()},
	}}
	uc := newCreateGroupUseCase(repo, memberRepo, nicknameResolver)

	input := validCreateInput(shared.RoleCoach)
	input.MemberNicknames = []string{"alice", "bob"}
	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Members) != 2 {
		t.Fatalf("expected 2 added members, got %d: %+v", len(out.Members), out.Members)
	}
	if len(memberRepo.savedMembers) != 2 {
		t.Fatalf("expected SaveAll to be called with 2 members, got %d", len(memberRepo.savedMembers))
	}
	for _, m := range out.Members {
		if m.Role != domainGroup.MemberRoleMember.String() {
			t.Errorf("expected role MEMBER, got %s", m.Role)
		}
	}
}

func TestCreateGroup_LeadNicknamesAddedSuccessfully(t *testing.T) {
	repo := &mockGroupRepository{}
	memberRepo := &mockMemberRepository{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"coach2": {ID: "lead-1", Nickname: "coach2", Name: "Coach Two", SystemRole: shared.RoleCoach.String()},
	}}
	uc := newCreateGroupUseCase(repo, memberRepo, nicknameResolver)

	input := validCreateInput(shared.RoleCoach)
	input.LeadNicknames = []string{"coach2"}
	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Members) != 1 || out.Members[0].Role != domainGroup.MemberRoleLead.String() {
		t.Fatalf("expected 1 added LEAD, got %+v", out.Members)
	}
}

func TestCreateGroup_ContestantAsLeadReturns400AndRollsBack(t *testing.T) {
	repo := &mockGroupRepository{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"contestant1": {ID: "c-1", Nickname: "contestant1", SystemRole: shared.RoleContestant.String()},
	}}
	uc := newCreateGroupUseCase(repo, nil, nicknameResolver)

	input := validCreateInput(shared.RoleCoach)
	input.LeadNicknames = []string{"contestant1"}
	_, err := uc.Execute(context.Background(), input)

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != domainGroup.ErrCodeInvalidLeadAssignment {
		t.Fatalf("expected INVALID_LEAD_ASSIGNMENT, got %v", err)
	}
	if repo.savedGroup != nil {
		t.Error("expected the group to never be persisted when a lead nickname is invalid")
	}
}

func TestCreateGroup_NicknameNotFoundReturns404AndRollsBack(t *testing.T) {
	repo := &mockGroupRepository{}
	uc := newCreateGroupUseCase(repo, nil, &mockNicknameResolver{}) // empty users map -> every lookup misses

	input := validCreateInput(shared.RoleCoach)
	input.MemberNicknames = []string{"ghost"}
	_, err := uc.Execute(context.Background(), input)

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != ErrCodeNicknameNotFound {
		t.Fatalf("expected NICKNAME_NOT_FOUND, got %v", err)
	}
	if repo.savedGroup != nil {
		t.Error("expected the group to never be persisted when a member nickname is not found")
	}
}

func TestCreateGroup_DuplicateNicknameInSameListDeduped(t *testing.T) {
	repo := &mockGroupRepository{}
	memberRepo := &mockMemberRepository{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "member-1", Nickname: "alice", SystemRole: shared.RoleContestant.String()},
	}}
	uc := newCreateGroupUseCase(repo, memberRepo, nicknameResolver)

	input := validCreateInput(shared.RoleCoach)
	input.MemberNicknames = []string{"alice", "alice"}
	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Members) != 1 {
		t.Fatalf("expected the repeated nickname to be deduped to 1 member, got %d", len(out.Members))
	}
}

func TestCreateGroup_SameUserInBothListsLeadWins(t *testing.T) {
	repo := &mockGroupRepository{}
	memberRepo := &mockMemberRepository{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "user-1", Nickname: "alice", SystemRole: shared.RoleCoach.String()},
	}}
	uc := newCreateGroupUseCase(repo, memberRepo, nicknameResolver)

	input := validCreateInput(shared.RoleCoach)
	input.MemberNicknames = []string{"alice"}
	input.LeadNicknames = []string{"alice"}
	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Members) != 1 || out.Members[0].Role != domainGroup.MemberRoleLead.String() {
		t.Fatalf("expected the user to be added once, as LEAD, got %+v", out.Members)
	}
}

func TestCreateGroup_CreatorOwnNicknameSilentlySkipped(t *testing.T) {
	repo := &mockGroupRepository{}
	memberRepo := &mockMemberRepository{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"me": {ID: "u1", Nickname: "me", SystemRole: shared.RoleCoach.String()},
	}}
	uc := newCreateGroupUseCase(repo, memberRepo, nicknameResolver)

	input := validCreateInput(shared.RoleCoach) // CurrentUser.ID == "u1"
	input.LeadNicknames = []string{"me"}
	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Members) != 0 {
		t.Errorf("expected the creator's own nickname to be skipped, got %+v", out.Members)
	}
	if len(memberRepo.savedMembers) != 0 {
		t.Errorf("expected SaveAll not to be called for the creator's own nickname, got %+v", memberRepo.savedMembers)
	}
}

func TestCreateGroup_SaveAllFailurePropagatesError(t *testing.T) {
	repo := &mockGroupRepository{}
	memberRepo := &mockMemberRepository{saveAllErr: apperror.NewInternal()}
	nicknameResolver := &mockNicknameResolver{users: map[string]*UserDisplay{
		"alice": {ID: "member-1", Nickname: "alice", SystemRole: shared.RoleContestant.String()},
	}}
	uc := newCreateGroupUseCase(repo, memberRepo, nicknameResolver)

	input := validCreateInput(shared.RoleCoach)
	input.MemberNicknames = []string{"alice"}
	_, err := uc.Execute(context.Background(), input)

	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeInternalError {
		t.Fatalf("expected INTERNAL_ERROR on SaveAll failure, got %v", err)
	}
}
