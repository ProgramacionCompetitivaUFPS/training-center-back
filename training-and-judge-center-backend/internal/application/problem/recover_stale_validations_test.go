package problem

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockStaleValidationRecoverer struct {
	recoverFn func(ctx context.Context, cutoff time.Time) (int, error)
}

func (m *mockStaleValidationRecoverer) RecoverStaleBefore(ctx context.Context, cutoff time.Time) (int, error) {
	if m.recoverFn != nil {
		return m.recoverFn(ctx, cutoff)
	}
	return 0, nil
}

func TestRecoverStaleValidations_PassesCutoffAndReturnsCount(t *testing.T) {
	var gotCutoff time.Time
	uc := NewRecoverStaleValidationsUseCase(&mockStaleValidationRecoverer{
		recoverFn: func(_ context.Context, cutoff time.Time) (int, error) {
			gotCutoff = cutoff
			return 2, nil
		},
	}, 20*time.Minute)

	lowerBound := time.Now().Add(-20 * time.Minute)
	count, err := uc.Execute(context.Background())
	upperBound := time.Now().Add(-20 * time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 recovered validations, got %d", count)
	}
	if gotCutoff.Before(lowerBound) || gotCutoff.After(upperBound) {
		t.Errorf("cutoff %v outside expected window [%v, %v]", gotCutoff, lowerBound, upperBound)
	}
}

func TestRecoverStaleValidations_PropagatesRecovererError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	uc := NewRecoverStaleValidationsUseCase(&mockStaleValidationRecoverer{
		recoverFn: func(_ context.Context, _ time.Time) (int, error) {
			return 0, wantErr
		},
	}, 20*time.Minute)

	if _, err := uc.Execute(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
}
