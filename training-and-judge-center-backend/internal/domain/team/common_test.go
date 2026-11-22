package team_test

import (
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var (
	fixedNow  = time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	plusTwo   = time.FixedZone("UTC+2", 2*3600)
	localNow  = time.Date(2026, 5, 4, 15, 0, 0, 0, plusTwo)
	utcOfNow  = time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC)
	emptyUser = shared.RestoreUserID("")
)

func mustTeamName(t *testing.T, s string) team.TeamName {
	t.Helper()
	n, err := team.NewTeamName(s)
	if err != nil {
		t.Fatalf("mustTeamName: %v", err)
	}
	return n
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
