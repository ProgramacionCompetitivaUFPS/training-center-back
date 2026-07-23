package submission

import (
	"context"
	"errors"
	"testing"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetSubmission_NotFound(t *testing.T) {
	repo := &mockSubmissionRepo{}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindNotFound {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestGetSubmission_AuthorCanViewPrivate(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Visibility != "PRIVATE" {
		t.Errorf("expected PRIVATE, got %s", out.Visibility)
	}
}

func TestGetSubmission_AdminCanViewPrivate(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsAdmin("admin-001"),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ID != testSubmissionID {
		t.Errorf("expected %s, got %s", testSubmissionID, out.ID)
	}
}

func TestGetSubmission_NonAuthor_PublicVisible(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", nil), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Visibility != "PUBLIC" {
		t.Errorf("expected PUBLIC, got %s", out.Visibility)
	}
}

func TestGetSubmission_NonAuthor_Private_NoContest_Forbidden(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
	if appErr.Code != domainSubmission.ErrCodeAccessDenied {
		t.Errorf("expected %s, got %s", domainSubmission.ErrCodeAccessDenied, appErr.Code)
	}
}

func TestGetSubmission_NonAuthor_Private_ContestLead_Allowed(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", &cid), nil
		},
	}
	lead := &mockLeadChecker{fn: func(contestID, userID string) (bool, error) {
		return contestID == testContestID && userID == testOtherUserID, nil
	}}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, lead)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGetSubmission_NonAuthor_Private_Teammate_Allowed(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", &cid), nil
		},
	}
	team := &mockTeamMembershipChecker{fn: func(contestID, viewerID, authorID string) (bool, error) {
		return true, nil
	}}
	uc := newGetUseCase(repo, nil, nil, nil, nil, team, nil)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGetSubmission_NonAuthor_Private_ContestButNotLeadOrTeammate_Forbidden(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", &cid), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestGetSubmission_SourceCodeReaderError_ReturnsError(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	reader := &mockSourceCodeReader{fn: func(_ string) ([]byte, error) {
		return nil, errors.New("storage error")
	}}
	uc := newGetUseCase(repo, reader, nil, nil, nil, nil, nil)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSubmission_OutputContainsContestWhenPresent(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", &cid), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  appshared.CurrentUser{ID: testAuthorID, Role: shared.RoleContestant},
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Contest == nil {
		t.Fatal("expected contest in output, got nil")
	}
	if out.Contest.ID != testContestID {
		t.Errorf("expected contest ID %s, got %s", testContestID, out.Contest.ID)
	}
}

func TestGetSubmission_OutputContestNilWhenNoContest(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", nil), nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Contest != nil {
		t.Errorf("expected nil contest, got %+v", out.Contest)
	}
}
