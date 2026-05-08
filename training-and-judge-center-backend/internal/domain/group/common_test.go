package group_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func assertValidationField(t *testing.T, label string, err error, wantField string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected validation error, got nil", label)
		return
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("%s: expected *apperror.AppError, got %T", label, err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("%s: Code = %q, want %q", label, appErr.Code, apperror.ErrCodeValidationError)
	}
	if len(appErr.Details) == 0 {
		t.Fatalf("%s: expected Details with field %q, got empty", label, wantField)
	}
	if appErr.Details[0].Field != wantField {
		t.Errorf("%s: Details[0].Field = %q, want %q", label, appErr.Details[0].Field, wantField)
	}
}
