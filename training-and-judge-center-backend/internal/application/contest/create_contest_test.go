package contest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func newCreateContestUseCase(
	repo *mockContestRepository,
	group *mockGroupProvider,
	member *mockGroupMemberProvider,
	problem *mockProblemProvider,
) *CreateContestUseCase {
	return NewCreateContestUseCase(repo, group, member, problem, stubOwner())
}

func validCreateInput() CreateContestInput {
	return CreateContestInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		Name:        "My Contest",
		StartTime:   testStart,
		EndTime:     testEnd,
	}
}

func TestCreateContest_SuccessByLead(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	out, err := uc.Execute(context.Background(), validCreateInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "My Contest" {
		t.Errorf("expected name 'My Contest', got %q", out.Name)
	}
	if out.Status != "SCHEDULED" {
		t.Errorf("expected SCHEDULED status, got %q", out.Status)
	}
	if out.Penalty != 20 {
		t.Errorf("expected default penalty 20, got %d", out.Penalty)
	}
}

func TestCreateContest_SuccessByAdmin(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), notLead(), defaultProblemProvider())

	in := validCreateInput()
	in.CurrentUser = asAdmin(callerID)

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("expected output, got nil")
	}
}

func TestCreateContest_SuccessWithProblems(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	in.Problems = []string{"problem-1", "problem-2"}

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Problems) != 2 {
		t.Errorf("expected 2 problems, got %d", len(out.Problems))
	}
	if out.ProblemCount != 2 {
		t.Errorf("expected problemCount 2, got %d", out.ProblemCount)
	}
	if out.Problems[0].Order != 1 || out.Problems[1].Order != 2 {
		t.Errorf("expected sequential orders 1,2 got %d,%d", out.Problems[0].Order, out.Problems[1].Order)
	}
}

func TestCreateContest_DeduplicatesProblems(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	in.Problems = []string{"problem-1", "problem-1", "problem-1"}

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Problems) != 1 {
		t.Errorf("expected 1 deduplicated problem, got %d", len(out.Problems))
	}
}

func TestCreateContest_GroupNotFound(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupNotFound(), isLead(), defaultProblemProvider())

	_, err := uc.Execute(context.Background(), validCreateInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeGroupNotFound {
		t.Errorf("expected GROUP_NOT_FOUND, got %v", err)
	}
}

func TestCreateContest_ForbiddenIfNotLead(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), notLead(), defaultProblemProvider())

	_, err := uc.Execute(context.Background(), validCreateInput())

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateContest_ForbiddenIfContestant(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), notLead(), defaultProblemProvider())

	in := validCreateInput()
	in.CurrentUser = asContestant(callerID)

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %v", err)
	}
}

func TestCreateContest_StartTimeInPast(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	past := time.Now().Add(-time.Hour)
	in.StartTime = past

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "START_TIME_IN_PAST" {
		t.Errorf("expected START_TIME_IN_PAST, got %v", err)
	}
}

func TestCreateContest_InvalidTimeRange(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	// Keep startTime valid (future), make endTime before startTime.
	in.EndTime = in.StartTime.Add(-time.Minute)

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "INVALID_TIME_RANGE" {
		t.Errorf("expected INVALID_TIME_RANGE, got %v", err)
	}
}

func TestCreateContest_ValidationErrorEmptyName(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	in.Name = ""

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestCreateContest_ProblemNotFound(t *testing.T) {
	pp := &mockProblemProvider{
		findBySlugsFn: func(_ context.Context, _ []string, _ string, _ bool) (map[string]*ProblemInfo, error) {
			return map[string]*ProblemInfo{}, nil // empty — slug absent
		},
	}
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), pp)

	in := validCreateInput()
	in.Problems = []string{"missing-problem"}

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeProblemNotFound {
		t.Errorf("expected PROBLEM_NOT_FOUND, got %v", err)
	}
}

func TestCreateContest_ProblemNotPublished(t *testing.T) {
	pp := &mockProblemProvider{
		findBySlugsFn: func(_ context.Context, slugs []string, _ string, _ bool) (map[string]*ProblemInfo, error) {
			result := make(map[string]*ProblemInfo)
			for _, s := range slugs {
				result[s] = &ProblemInfo{ID: testProblemID, Slug: s, Title: s, IsPublished: false, CanAdd: true}
			}
			return result, nil
		},
	}
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), pp)

	in := validCreateInput()
	in.Problems = []string{"draft-problem"}

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeProblemNotPublished {
		t.Errorf("expected PROBLEM_NOT_PUBLISHED, got %v", err)
	}
}

func TestCreateContest_ProblemAccessDenied(t *testing.T) {
	pp := &mockProblemProvider{
		findBySlugsFn: func(_ context.Context, slugs []string, _ string, _ bool) (map[string]*ProblemInfo, error) {
			result := make(map[string]*ProblemInfo)
			for _, s := range slugs {
				result[s] = &ProblemInfo{ID: testProblemID, Slug: s, Title: s, IsPublished: true, CanAdd: false}
			}
			return result, nil
		},
	}
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), pp)

	in := validCreateInput()
	in.Problems = []string{"private-problem"}

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != ErrCodeProblemAccessDenied {
		t.Errorf("expected PROBLEM_ACCESS_DENIED, got %v", err)
	}
}

func TestCreateContest_DefaultPenalty(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	out, err := uc.Execute(context.Background(), validCreateInput())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Penalty != 20 {
		t.Errorf("expected default penalty 20, got %d", out.Penalty)
	}
}

func TestCreateContest_CustomPenalty(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	p := 30
	in.Penalty = &p

	out, err := uc.Execute(context.Background(), in)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Penalty != 30 {
		t.Errorf("expected penalty 30, got %d", out.Penalty)
	}
}

func TestCreateContest_InvalidPenalty(t *testing.T) {
	uc := newCreateContestUseCase(&mockContestRepository{}, groupFound(), isLead(), defaultProblemProvider())

	in := validCreateInput()
	p := -1
	in.Penalty = &p

	_, err := uc.Execute(context.Background(), in)

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR for negative penalty, got %v", err)
	}
}
