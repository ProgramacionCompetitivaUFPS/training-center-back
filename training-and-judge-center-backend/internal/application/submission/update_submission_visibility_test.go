package submission

import (
	"context"
	"errors"
	"testing"

	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/testutil"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUpdateVisibility_SubmissionNotFound(t *testing.T) {
	repo := &mockSubmissionRepo{}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsContestant(testAuthorID),
		SubmissionID:  testSubmissionID,
		NewVisibility: "PUBLIC",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestUpdateVisibility_NonAuthorForbidden(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsContestant(testOtherUserID),
		SubmissionID:  testSubmissionID,
		NewVisibility: "PUBLIC",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
	if appErr.Code != domainSubmission.ErrCodeAccessDenied {
		t.Errorf("expected %s, got %s", domainSubmission.ErrCodeAccessDenied, appErr.Code)
	}
}

func TestUpdateVisibility_InvalidVisibilityReturns400(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsContestant(testAuthorID),
		SubmissionID:  testSubmissionID,
		NewVisibility: "INVALID",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateVisibility_SetPublic_Succeeds(t *testing.T) {
	var updated *domainSubmission.Submission
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
		updateVisFn: func(s *domainSubmission.Submission) error {
			updated = s
			return nil
		},
	}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsContestant(testAuthorID),
		SubmissionID:  testSubmissionID,
		NewVisibility: "PUBLIC",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated == nil {
		t.Fatal("expected UpdateVisibility to be called")
	}
	if !updated.Visibility().IsPublic() {
		t.Errorf("expected visibility PUBLIC, got %s", updated.Visibility().String())
	}
}

func TestUpdateVisibility_SetPrivate_Succeeds(t *testing.T) {
	var updated *domainSubmission.Submission
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", nil), nil
		},
		updateVisFn: func(s *domainSubmission.Submission) error {
			updated = s
			return nil
		},
	}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsContestant(testAuthorID),
		SubmissionID:  testSubmissionID,
		NewVisibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated == nil {
		t.Fatal("expected UpdateVisibility to be called")
	}
	if !updated.Visibility().IsPrivate() {
		t.Errorf("expected visibility PRIVATE, got %s", updated.Visibility().String())
	}
}

func TestUpdateVisibility_RepoError_Propagates(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
		updateVisFn: func(_ *domainSubmission.Submission) error {
			return apperror.NewInternal()
		},
	}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsContestant(testAuthorID),
		SubmissionID:  testSubmissionID,
		NewVisibility: "PUBLIC",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestUpdateVisibility_AdminForbidden(t *testing.T) {
	// Only the author can change visibility; admin is not special here.
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	uc := newUpdateVisUseCase(repo)

	err := uc.Execute(context.Background(), UpdateSubmissionVisibilityInput{
		CurrentUser:   testutil.AsAdmin("admin-001"),
		SubmissionID:  testSubmissionID,
		NewVisibility: "PUBLIC",
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}
