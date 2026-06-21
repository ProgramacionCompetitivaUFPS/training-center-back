package submission

import (
	"context"
	"testing"

	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/testutil"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestListProblemSubmissions_NonAdmin_OnlyPublic(t *testing.T) {
	var capturedFilters domainsubmission.ListFilters
	repo := &mockSubmissionRepo{
		listFn: func(f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			capturedFilters = f
			return []*domainsubmission.Submission{}, 0, nil
		},
	}
	uc := newListProblemSubmissionsUseCase(repo, publicProblem(), nil)
	_, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: asContestant(testUserID),
		ProblemSlug: testProblemSlug,
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.Visibility == nil || *capturedFilters.Visibility != "PUBLIC" {
		t.Errorf("expected Visibility filter = PUBLIC, got %v", capturedFilters.Visibility)
	}
	if !capturedFilters.ExcludeContest {
		t.Error("expected ExcludeContest = true")
	}
}

func TestListProblemSubmissions_Admin_SeesAll(t *testing.T) {
	var capturedFilters domainsubmission.ListFilters
	repo := &mockSubmissionRepo{
		listFn: func(f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			capturedFilters = f
			return []*domainsubmission.Submission{}, 0, nil
		},
	}
	uc := newListProblemSubmissionsUseCase(repo, publicProblem(), nil)
	_, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: asAdmin(testUserID),
		ProblemSlug: testProblemSlug,
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.Visibility != nil {
		t.Errorf("admin should not have visibility filter, got %v", *capturedFilters.Visibility)
	}
	if !capturedFilters.ExcludeContest {
		t.Error("expected ExcludeContest = true")
	}
}

func TestListProblemSubmissions_OnlyMine_SetsUserIDFilter(t *testing.T) {
	var capturedFilters domainsubmission.ListFilters
	repo := &mockSubmissionRepo{
		listFn: func(f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			capturedFilters = f
			return []*domainsubmission.Submission{}, 0, nil
		},
	}
	uc := newListProblemSubmissionsUseCase(repo, publicProblem(), nil)
	_, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: asContestant(testUserID),
		ProblemSlug: testProblemSlug,
		OnlyMine:    true,
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.UserID == nil {
		t.Fatal("expected UserID filter to be set for OnlyMine=true")
	}
	if capturedFilters.UserID.String() != testUserID {
		t.Errorf("UserID = %q, want %q", capturedFilters.UserID.String(), testUserID)
	}
	// OnlyMine takes precedence — no visibility filter
	if capturedFilters.Visibility != nil {
		t.Errorf("OnlyMine should not add visibility filter, got %v", *capturedFilters.Visibility)
	}
}

func TestListProblemSubmissions_ProblemNotFound_Returns404(t *testing.T) {
	repo := &mockSubmissionRepo{}
	problem := &mockProblemProvider{
		fn: func(_ string) (*ProblemInfo, error) {
			return nil, apperror.NewNotFound("PROBLEM_NOT_FOUND", "not found")
		},
	}
	uc := newListProblemSubmissionsUseCase(repo, problem, nil)
	_, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: asContestant(testUserID),
		ProblemSlug: "ghost",
		Page:        1,
		Limit:       20,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if ae.Kind != apperror.KindNotFound {
		t.Errorf("Kind = %q, want NOT_FOUND", ae.Kind)
	}
}

func TestListProblemSubmissions_ProblemInfoPropagatedToSummary(t *testing.T) {
	sub := aSubmission(testOtherUserID, "PUBLIC", nil)
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{sub}, 1, nil
		},
	}
	uc := newListProblemSubmissionsUseCase(repo, publicProblem(), &mockUserProvider{})
	out, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: testutil.AsAdmin(testUserID),
		ProblemSlug: testProblemSlug,
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(out.Submissions))
	}
	got := out.Submissions[0]
	if got.Problem.Slug != testProblemSlug {
		t.Errorf("Problem.Slug = %q, want %q", got.Problem.Slug, testProblemSlug)
	}
	if got.Contest != nil {
		t.Error("contest should be nil for problem submissions outside contest")
	}
}
