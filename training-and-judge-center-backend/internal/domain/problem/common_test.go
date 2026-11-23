package problem_test

import (
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

const (
	testAuthorID   = "aaaaaaaa-0000-0000-0000-000000000001"
	testModifierID = "aaaaaaaa-0000-0000-0000-000000000002"
	testStrangerID = "aaaaaaaa-0000-0000-0000-000000000003"
	testProblemID  = "bbbbbbbb-0000-0000-0000-000000000001"
)

func intPtr(v int) *int { return &v }

func detailFields(t *testing.T, err error) []string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected a validation apperror, got %v", err)
	}
	fields := make([]string, len(appErr.Details))
	for i, d := range appErr.Details {
		fields[i] = d.Field
	}
	return fields
}

func assertInternalError(t *testing.T, label string, err error) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Errorf("%s: expected an internal apperror, got %v", label, err)
	}
}
