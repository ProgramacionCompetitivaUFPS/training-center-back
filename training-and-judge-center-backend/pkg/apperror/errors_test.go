package apperror_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestConstructors_SetKindCodeAndMessage(t *testing.T) {
	tests := []struct {
		name        string
		err         *apperror.AppError
		wantKind    apperror.Kind
		wantCode    string
		wantMessage string
	}{
		{"bad request", apperror.NewBadRequest("BAD_THING", "bad thing"), apperror.KindBadRequest, "BAD_THING", "bad thing"},
		{"conflict", apperror.NewConflict("TAKEN", "already taken"), apperror.KindConflict, "TAKEN", "already taken"},
		{"not found", apperror.NewNotFound("GONE", "it is gone"), apperror.KindNotFound, "GONE", "it is gone"},
		{"unauthorized", apperror.NewUnauthorized("NO_TOKEN", "log in first"), apperror.KindUnauthorized, "NO_TOKEN", "log in first"},
		{"forbidden", apperror.NewForbidden("NOT_YOURS", "not yours"), apperror.KindForbidden, "NOT_YOURS", "not yours"},
		{"service unavailable", apperror.NewServiceUnavailable("BUSY", "try later"), apperror.KindServiceUnavailable, "BUSY", "try later"},
		{"internal never leaks details", apperror.NewInternal(), apperror.KindInternal, apperror.ErrCodeInternalError, "An unexpected error occurred"},
		{"payload too large", apperror.NewPayloadTooLarge("max 5 MB"), apperror.KindPayloadTooLarge, apperror.ErrCodePayloadTooLarge, "max 5 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantKind, tt.err.Kind)
			assert.Equal(t, tt.wantCode, tt.err.Code)
			assert.Equal(t, tt.wantMessage, tt.err.Error())
			assert.Empty(t, tt.err.Details)
		})
	}
}

func TestNewValidation_CarriesTheFieldErrors(t *testing.T) {
	details := []apperror.FieldError{{Field: "email", Message: "invalid"}, {Field: "name", Code: "REQUIRED", Message: "required"}}

	err := apperror.NewValidation(details)

	assert.Equal(t, apperror.KindValidation, err.Kind)
	assert.Equal(t, apperror.ErrCodeValidationError, err.Code)
	assert.Equal(t, "Invalid request data", err.Error())
	assert.Equal(t, details, err.Details)
}

func TestNewConflictWithDetails_KeepsTheFieldLevelDetails(t *testing.T) {
	details := []apperror.FieldError{{Field: "nickname", Message: "nickname already in use"}}

	err := apperror.NewConflictWithDetails("NICKNAME_CONFLICT", "nickname already in use", details)

	assert.Equal(t, apperror.KindConflict, err.Kind)
	assert.Equal(t, "NICKNAME_CONFLICT", err.Code)
	assert.Equal(t, details, err.Details)
}

func TestNewTooManyRequests_FillsDefaultsOnlyWhenEmpty(t *testing.T) {
	defaults := apperror.NewTooManyRequests("", "", 30)
	assert.Equal(t, apperror.KindTooManyRequests, defaults.Kind)
	assert.Equal(t, "RATE_LIMIT_EXCEEDED", defaults.Code)
	assert.Equal(t, "Too many requests. Please try again later", defaults.Message)
	assert.Equal(t, 30, defaults.RetryAfter)

	custom := apperror.NewTooManyRequests("LOGIN_LOCKED", "locked for a while", 600)
	assert.Equal(t, "LOGIN_LOCKED", custom.Code)
	assert.Equal(t, "locked for a while", custom.Message)
	assert.Equal(t, 600, custom.RetryAfter)
}

func TestWithCause_UnwrapsToTheOriginalError(t *testing.T) {
	cause := errors.New("connection reset")

	err := apperror.NewInternal().WithCause(cause)

	assert.ErrorIs(t, err, cause)
	assert.Equal(t, "An unexpected error occurred", err.Error(), "the cause is never shown to the client")
	assert.NoError(t, errors.Unwrap(apperror.NewInternal()))
}

func TestAccumulateFieldErrors(t *testing.T) {
	t.Run("nil error accumulates nothing", func(t *testing.T) {
		var collected []apperror.FieldError

		err := apperror.AccumulateFieldErrors(nil, &collected)

		require.NoError(t, err)
		assert.Empty(t, collected)
	})

	t.Run("an app error contributes its details and is swallowed", func(t *testing.T) {
		collected := []apperror.FieldError{{Field: "a", Message: "first"}}
		validation := apperror.NewValidation([]apperror.FieldError{{Field: "b", Message: "second"}, {Field: "c", Message: "third"}})

		err := apperror.AccumulateFieldErrors(validation, &collected)

		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, []string{collected[0].Field, collected[1].Field, collected[2].Field})
	})

	t.Run("a wrapped app error is still recognised", func(t *testing.T) {
		var collected []apperror.FieldError
		wrapped := errors.Join(errors.New("context"), apperror.NewValidation([]apperror.FieldError{{Field: "x", Message: "bad"}}))

		err := apperror.AccumulateFieldErrors(wrapped, &collected)

		require.NoError(t, err)
		require.Len(t, collected, 1)
	})

	t.Run("an unknown error becomes an internal error and adds nothing", func(t *testing.T) {
		var collected []apperror.FieldError

		err := apperror.AccumulateFieldErrors(errors.New("disk full"), &collected)

		var appErr *apperror.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, apperror.KindInternal, appErr.Kind)
		assert.Empty(t, collected)
	})
}
