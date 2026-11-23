package user_test

import (
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

var (
	plusTwo  = time.FixedZone("UTC+2", 2*3600)
	localNow = time.Date(2024, 1, 15, 14, 0, 0, 0, plusTwo)
)

func makeTestUser(t *testing.T) *user.User {
	t.Helper()
	email, _ := user.NewEmail("test@example.com")
	password, _ := user.NewPassword("SecurePass1!")
	nickname, _ := user.NewNickname("user-test")
	u, err := user.NewUser("user-id", testNow, email, password, "John Doe", nickname, "Colombia", "Bogotá", "UFPS")
	if err != nil {
		t.Fatalf("makeTestUser: %v", err)
	}
	return u
}

func strPtr(s string) *string { return &s }

func mustEmail(t *testing.T, s string) user.Email {
	t.Helper()
	e, err := user.NewEmail(s)
	if err != nil {
		t.Fatalf("mustEmail: %v", err)
	}
	return e
}

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

func assertErrorCode(t *testing.T, label string, err error, wantCode string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("%s: expected *apperror.AppError, got %T: %v", label, err, err)
		return
	}
	if appErr.Code != wantCode {
		t.Errorf("%s: Code = %q, want %q", label, appErr.Code, wantCode)
	}
}

func assertInternalError(t *testing.T, label string, err error) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Errorf("%s: expected an internal apperror, got %v", label, err)
	}
}

func assertInUTC(t *testing.T, label string, got, want time.Time) {
	t.Helper()
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("%s = %v, want %v in UTC", label, got, want)
	}
}
