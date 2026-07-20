package judge

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRecoverStaleSubmissions_PassesCutoffAndReturnsCount(t *testing.T) {
	var gotCutoff time.Time
	uc := NewRecoverStaleSubmissionsUseCase(&mockStaleSubmissionRecoverer{
		recoverFn: func(_ context.Context, cutoff time.Time) (int, error) {
			gotCutoff = cutoff
			return 3, nil
		},
	}, 10*time.Minute)

	lowerBound := time.Now().Add(-10 * time.Minute)
	count, err := uc.Execute(context.Background())
	upperBound := time.Now().Add(-10 * time.Minute)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 recovered submissions, got %d", count)
	}
	if gotCutoff.Before(lowerBound) || gotCutoff.After(upperBound) {
		t.Errorf("cutoff %v outside expected window [%v, %v]", gotCutoff, lowerBound, upperBound)
	}
}

func TestRecoverStaleSubmissions_PropagatesRecovererError(t *testing.T) {
	wantErr := errors.New("db unavailable")
	uc := NewRecoverStaleSubmissionsUseCase(&mockStaleSubmissionRecoverer{
		recoverFn: func(_ context.Context, _ time.Time) (int, error) {
			return 0, wantErr
		},
	}, 10*time.Minute)

	if _, err := uc.Execute(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
}
