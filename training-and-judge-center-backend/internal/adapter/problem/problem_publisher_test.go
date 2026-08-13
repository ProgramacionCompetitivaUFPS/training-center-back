package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testPublishNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func TestMarkPublished_Success(t *testing.T) {
	publisher := NewProblemPublisher(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	})

	if err := publisher.MarkPublished(context.Background(), testValidationProblemID, testPublishNow); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMarkPublished_NoRowsAffected_ReturnsNotFound(t *testing.T) {
	publisher := NewProblemPublisher(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	})

	err := publisher.MarkPublished(context.Background(), testValidationProblemID, testPublishNow)
	assertAppErrorKind(t, err, apperror.KindNotFound)

	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Code != domainProblem.ErrCodeProblemNotFound {
		t.Errorf("Code: got %q, want %q", appErr.Code, domainProblem.ErrCodeProblemNotFound)
	}
}

func TestMarkPublished_DBError_ReturnsInternal(t *testing.T) {
	publisher := NewProblemPublisher(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := publisher.MarkPublished(context.Background(), testValidationProblemID, testPublishNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}
