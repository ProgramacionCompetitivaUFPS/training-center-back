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

func TestGetSubmission_SourceCodeMissing_ReturnsDetailWithoutSource(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	reader := &mockSourceCodeReader{fn: func(_ string) ([]byte, error) {
		return nil, apperror.NewNotFound(ErrCodeSourceCodeNotFound, "source code not found")
	}}
	uc := newGetUseCase(repo, reader, nil, nil, nil, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SourceCode != nil {
		t.Errorf("SourceCode = %q, want nil", *out.SourceCode)
	}
	if out.ID != testSubmissionID {
		t.Errorf("ID = %q, want %q", out.ID, testSubmissionID)
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

func TestGetSubmission_OutputContainsContestGroupAndProblemPosition(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", &cid), nil
		},
	}
	cont := &mockContestProvider{
		positionFn: func(contestID, _ string) (*int, error) {
			if contestID != testContestID {
				t.Errorf("position looked up in contest %s, want %s", contestID, testContestID)
			}
			position := 3
			return &position, nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, cont, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Contest.GroupID != "group-1" {
		t.Errorf("contest group ID = %q, want group-1", out.Contest.GroupID)
	}
	if out.ProblemPosition == nil || *out.ProblemPosition != 3 {
		t.Errorf("problem position = %v, want 3", out.ProblemPosition)
	}
}

func TestGetSubmission_ProblemPositionNilWhenProblemDroppedFromContest(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", &cid), nil
		},
	}
	cont := &mockContestProvider{
		positionFn: func(_, _ string) (*int, error) { return nil, nil },
	}
	uc := newGetUseCase(repo, nil, nil, nil, cont, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("a problem dropped from the contest must not fail the detail, got %v", err)
	}
	if out.ProblemPosition != nil {
		t.Errorf("problem position = %d, want nil", *out.ProblemPosition)
	}
}

func TestGetSubmission_ProblemPositionNilOutsideContest(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", nil), nil
		},
	}
	cont := &mockContestProvider{
		positionFn: func(_, _ string) (*int, error) {
			t.Error("position must not be looked up for a submission outside a contest")
			return nil, nil
		},
	}
	uc := newGetUseCase(repo, nil, nil, nil, cont, nil, nil)

	out, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.ProblemPosition != nil {
		t.Errorf("problem position = %d, want nil", *out.ProblemPosition)
	}
}

func TestGetSubmission_ProblemPositionErrorPropagates(t *testing.T) {
	cid := testContestID
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", &cid), nil
		},
	}
	boom := errors.New("db down")
	cont := &mockContestProvider{
		positionFn: func(_, _ string) (*int, error) { return nil, boom },
	}
	uc := newGetUseCase(repo, nil, nil, nil, cont, nil, nil)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected the provider error, got %v", err)
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

func TestGetSubmission_OutputContainsCompileLog(t *testing.T) {
	logText := "main.cpp:3:5: error: expected ';'"
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			sub := aSubmission(testAuthorID, "PUBLIC", nil)
			if err := sub.Start(testNow); err != nil {
				t.Fatalf("setup: %v", err)
			}
			if err := sub.MarkCompilationError(logText, testNow); err != nil {
				t.Fatalf("setup: %v", err)
			}
			return sub, nil
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
	if out.CompileLog == nil || *out.CompileLog != logText {
		t.Errorf("CompileLog = %v, want %q", out.CompileLog, logText)
	}
}

func TestGetSubmission_CompileLogNilWhenNotCompilationError(t *testing.T) {
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
	if out.CompileLog != nil {
		t.Errorf("CompileLog = %q, want nil", *out.CompileLog)
	}
}

func TestGetSubmission_NonAuthor_Private_NoContest_LeadOfAuthor_Allowed(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	lead := &mockLeadChecker{memberFn: func(leadID, memberID string) (bool, error) {
		return leadID == testOtherUserID && memberID == testAuthorID, nil
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

func TestGetSubmission_NonAuthor_Private_NotLeadOfAuthor_Forbidden(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	lead := &mockLeadChecker{memberFn: func(_, _ string) (bool, error) { return false, nil }}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, lead)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindForbidden {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestGetSubmission_LeadOfMemberCheckError_ReturnsError(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	lead := &mockLeadChecker{memberFn: func(_, _ string) (bool, error) { return false, errors.New("db down") }}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, lead)

	_, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testOtherUserID),
		SubmissionID: testSubmissionID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSubmission_LeadOfMemberNotConsultedForAuthor(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainSubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	lead := &mockLeadChecker{memberFn: func(_, _ string) (bool, error) {
		t.Error("lead check must not run for the author")
		return false, nil
	}}
	uc := newGetUseCase(repo, nil, nil, nil, nil, nil, lead)

	if _, err := uc.Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
