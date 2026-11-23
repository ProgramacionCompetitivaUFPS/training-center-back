package shared_test

import (
	"errors"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func assertBadRequest(t *testing.T, label string, err error) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Errorf("%s: expected *apperror.AppError, got %T: %v", label, err, err)
		return
	}
	if appErr.Kind != apperror.KindBadRequest || appErr.Code != apperror.ErrCodeBadRequest {
		t.Errorf("%s: kind/code = %s/%s, want %s/%s", label, appErr.Kind, appErr.Code, apperror.KindBadRequest, apperror.ErrCodeBadRequest)
	}
}
