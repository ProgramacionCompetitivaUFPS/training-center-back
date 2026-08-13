package problem

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestRecoverStaleBefore_Success_ReturnsCount(t *testing.T) {
	recoverer := NewStaleValidationRecoverer(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 3"), nil
		},
	})

	count, err := recoverer.RecoverStaleBefore(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("count: got %d, want 3", count)
	}
}

func TestRecoverStaleBefore_NoneStale_ReturnsZero(t *testing.T) {
	recoverer := NewStaleValidationRecoverer(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	})

	count, err := recoverer.RecoverStaleBefore(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("count: got %d, want 0", count)
	}
}

func TestRecoverStaleBefore_DBError_ReturnsInternal(t *testing.T) {
	recoverer := NewStaleValidationRecoverer(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	_, err := recoverer.RecoverStaleBefore(context.Background(), time.Now())
	assertAppErrorKind(t, err, apperror.KindInternal)
}
