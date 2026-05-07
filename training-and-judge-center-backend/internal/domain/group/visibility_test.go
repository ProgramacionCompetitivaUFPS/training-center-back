package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

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

func TestNewVisibility(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantOK    bool
		wantValue group.Visibility
	}{
		{"visible", "VISIBLE", true, group.VisibilityVisible},
		{"not visible", "NOT_VISIBLE", true, group.VisibilityNotVisible},
		{"lowercase rejected", "visible", false, group.Visibility{}},
		{"empty string rejected", "", false, group.Visibility{}},
		{"hidden rejected", "HIDDEN", false, group.Visibility{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := group.NewVisibility(tc.input)
			if tc.wantOK {
				if err != nil {
					t.Errorf("NewVisibility(%q) unexpected error: %v", tc.input, err)
				}
				if got != tc.wantValue {
					t.Errorf("NewVisibility(%q) = %q, want %q", tc.input, got, tc.wantValue)
				}
			} else {
				assertValidationField(t, "NewVisibility("+tc.input+")", err, "visibility")
			}
		})
	}
}
