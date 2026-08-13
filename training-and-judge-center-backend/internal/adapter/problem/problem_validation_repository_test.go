package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testValidationNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

const (
	testValidationID        = "validation-aaaa-0001"
	testValidationProblemID = "bbbbbbbb-0000-0000-0000-000000000001"
	testValidationUserID    = "cccccccc-0000-0000-0000-000000000001"
)

func pendingValidation() *domainProblem.ProblemValidation {
	v, err := domainProblem.NewProblemValidation(testValidationID, testValidationProblemID, shared.RestoreUserID(testValidationUserID), testValidationNow)
	if err != nil {
		panic(err)
	}
	return v
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestSave_Success(t *testing.T) {
	repo := NewProblemValidationRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	})

	if err := repo.Save(context.Background(), pendingValidation()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSave_DBError_ReturnsInternal(t *testing.T) {
	repo := NewProblemValidationRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.Save(context.Background(), pendingValidation())
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestSave_UniqueViolation_ReturnsConflict(t *testing.T) {
	repo := NewProblemValidationRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, &pgconn.PgError{
				Code:           "23505",
				ConstraintName: "idx_problem_validations_active_per_problem",
			}
		},
	})

	err := repo.Save(context.Background(), pendingValidation())
	assertAppErrorKind(t, err, apperror.KindConflict)

	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("Code: got %q, want %q", appErr.Code, domainProblem.ErrCodeValidationInProgress)
	}
}

// ── FindByID ──────────────────────────────────────────────────────────────────

func TestFindByID_Success(t *testing.T) {
	completedAt := testValidationNow.Add(time.Minute)
	result := `{"validationLogs":["ok"]}`

	repo := NewProblemValidationRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*string)) = testValidationProblemID
				*(dest[1].(*string)) = testValidationUserID
				*(dest[2].(*string)) = "PASSED"
				*(dest[3].(*time.Time)) = testValidationNow
				*(dest[4].(**time.Time)) = &completedAt
				*(dest[5].(**string)) = &result
				return nil
			}}
		},
	})

	v, err := repo.FindByID(context.Background(), testValidationID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.ID() != testValidationID {
		t.Errorf("ID(): got %q, want %q", v.ID(), testValidationID)
	}
	if v.ProblemID() != testValidationProblemID {
		t.Errorf("ProblemID(): got %q, want %q", v.ProblemID(), testValidationProblemID)
	}
	if !v.Status().IsPassed() {
		t.Error("Status(): expected IsPassed() true")
	}
	if v.CompletedAt() == nil || !v.CompletedAt().Equal(completedAt) {
		t.Errorf("CompletedAt(): got %v, want %v", v.CompletedAt(), completedAt)
	}
	if v.ResultJSON() == nil || *v.ResultJSON() != result {
		t.Errorf("ResultJSON(): got %v, want %q", v.ResultJSON(), result)
	}
}

func TestFindByID_NotFound_ReturnsNotFound(t *testing.T) {
	repo := NewProblemValidationRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := repo.FindByID(context.Background(), testValidationID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

// ── FindLatestByProblemID ─────────────────────────────────────────────────────

func TestFindLatestByProblemID_Found(t *testing.T) {
	repo := NewProblemValidationRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*string)) = testValidationID
				*(dest[1].(*string)) = testValidationUserID
				*(dest[2].(*string)) = "RUNNING"
				*(dest[3].(*time.Time)) = testValidationNow
				*(dest[4].(**time.Time)) = nil
				*(dest[5].(**string)) = nil
				return nil
			}}
		},
	})

	v, found, err := repo.FindLatestByProblemID(context.Background(), testValidationProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("found: got false, want true")
	}
	if !v.Status().IsRunning() {
		t.Error("Status(): expected IsRunning() true")
	}
}

func TestFindLatestByProblemID_NotFound_ReturnsFalseNoError(t *testing.T) {
	repo := NewProblemValidationRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	v, found, err := repo.FindLatestByProblemID(context.Background(), testValidationProblemID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found {
		t.Error("found: got true, want false")
	}
	if v != nil {
		t.Errorf("expected nil ProblemValidation, got %v", v)
	}
}
