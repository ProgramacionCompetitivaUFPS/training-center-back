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

// TestListProblemSubmissions_ManyRowsSameUser_BatchesUserLookup is the regression guard for
// the N+1 fix: several submissions from the same user (realistic — someone resubmitting to a
// popular problem) must all resolve correctly while GetUsersByIDs is called exactly once, not
// once per row (and not once per row even though every row shares one user ID).
func TestListProblemSubmissions_ManyRowsSameUser_BatchesUserLookup(t *testing.T) {
	subs := []*domainsubmission.Submission{
		aSubmission(testOtherUserID, "PUBLIC", nil),
		aSubmission(testOtherUserID, "PUBLIC", nil),
		aSubmission(testOtherUserID, "PUBLIC", nil),
	}
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return subs, len(subs), nil
		},
	}
	usr := &mockUserProvider{}
	uc := newListProblemSubmissionsUseCase(repo, publicProblem(), usr)
	out, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: testutil.AsAdmin(testUserID),
		ProblemSlug: testProblemSlug,
		Page:        1,
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 3 {
		t.Fatalf("expected 3 submissions, got %d", len(out.Submissions))
	}
	for i, s := range out.Submissions {
		if s.SubmittedBy.ID != testOtherUserID {
			t.Errorf("Submissions[%d].SubmittedBy.ID = %q, want %q", i, s.SubmittedBy.ID, testOtherUserID)
		}
	}
	if usr.batchCalls != 1 {
		t.Errorf("GetUsersByIDs called %d times, want exactly 1 (batched, deduped)", usr.batchCalls)
	}
}

// TestListProblemSubmissions_UserMissingFromBatch_Returns404 mirrors what the single-row
// GetUserByID would do if the user didn't exist — user_id is NOT NULL and FK-backed, so this
// is a defensive data-consistency check, not a normal path.
func TestListProblemSubmissions_UserMissingFromBatch_Returns404(t *testing.T) {
	sub := aSubmission(testOtherUserID, "PUBLIC", nil)
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{sub}, 1, nil
		},
	}
	usr := &mockUserProvider{
		fn: func(_ string) (*UserDisplay, error) {
			return nil, apperror.NewNotFound("USER_NOT_FOUND", "not found")
		},
	}
	uc := newListProblemSubmissionsUseCase(repo, publicProblem(), usr)
	_, err := uc.Execute(context.Background(), ListProblemSubmissionsInput{
		CurrentUser: testutil.AsAdmin(testUserID),
		ProblemSlug: testProblemSlug,
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
