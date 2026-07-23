package submission

import (
	"context"
	"testing"

	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestListMySubmissions_ReturnsOwnSubmissions(t *testing.T) {
	sub := aSubmission(testAuthorID, "PRIVATE", nil)
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{sub}, 1, nil
		},
	}
	uc := newListMyUseCase(repo, nil, nil, nil, nil)
	out, err := uc.Execute(context.Background(), ListMySubmissionsInput{
		CurrentUser: callerUser(),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(out.Submissions))
	}
	if out.Total != 1 {
		t.Errorf("Total = %d, want 1", out.Total)
	}
	got := out.Submissions[0]
	if got.ID != testSubmissionID {
		t.Errorf("ID = %q, want %q", got.ID, testSubmissionID)
	}
	if got.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", got.Status)
	}
	if got.Visibility != "PRIVATE" {
		t.Errorf("Visibility = %q, want PRIVATE", got.Visibility)
	}
	if got.SubmittedBy.ID != testAuthorID {
		t.Errorf("SubmittedBy.ID = %q, want %q", got.SubmittedBy.ID, testAuthorID)
	}
}

func TestListMySubmissions_EmptyResult(t *testing.T) {
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{}, 0, nil
		},
	}
	uc := newListMyUseCase(repo, nil, nil, nil, nil)
	out, err := uc.Execute(context.Background(), ListMySubmissionsInput{
		CurrentUser: callerUser(),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 0 {
		t.Errorf("expected 0 submissions, got %d", len(out.Submissions))
	}
}

func TestListMySubmissions_ProblemSlugFilter_ResolvesToProblemID(t *testing.T) {
	var capturedFilters domainsubmission.ListFilters
	repo := &mockSubmissionRepo{
		listFn: func(f domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			capturedFilters = f
			return []*domainsubmission.Submission{}, 0, nil
		},
	}
	slug := testProblemSlug
	uc := newListMyUseCase(repo, nil, nil, nil, publicProblem())
	_, err := uc.Execute(context.Background(), ListMySubmissionsInput{
		CurrentUser: callerUser(),
		ProblemSlug: &slug,
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters.ProblemID == nil {
		t.Fatal("expected ProblemID filter to be set")
	}
	if *capturedFilters.ProblemID != testProblemID {
		t.Errorf("ProblemID = %q, want %q", *capturedFilters.ProblemID, testProblemID)
	}
}

func TestListMySubmissions_ProblemSlugNotFound_Returns404(t *testing.T) {
	repo := &mockSubmissionRepo{}
	problem := &mockProblemProvider{
		fn: func(_ string) (*ProblemInfo, error) {
			return nil, apperror.NewNotFound("PROBLEM_NOT_FOUND", "not found")
		},
	}
	slug := "ghost"
	uc := newListMyUseCase(repo, nil, nil, nil, problem)
	_, err := uc.Execute(context.Background(), ListMySubmissionsInput{
		CurrentUser: callerUser(),
		ProblemSlug: &slug,
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

func TestListMySubmissions_ContestSubmission_ContestIncluded(t *testing.T) {
	cid := testContestID
	sub := aSubmission(testAuthorID, "PRIVATE", &cid)
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{sub}, 1, nil
		},
	}
	uc := newListMyUseCase(repo, nil, nil, &mockContestProvider{}, nil)
	out, err := uc.Execute(context.Background(), ListMySubmissionsInput{
		CurrentUser: callerUser(),
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Submissions[0].Contest == nil {
		t.Error("expected contest in summary, got nil")
	}
	if out.Submissions[0].Contest.ID != testContestID {
		t.Errorf("Contest.ID = %q, want %q", out.Submissions[0].Contest.ID, testContestID)
	}
}
