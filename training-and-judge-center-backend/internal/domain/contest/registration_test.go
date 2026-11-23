package contest_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/contest"
)

func TestNewContestRegistration_StoresTheFieldsAndTheInstantInUTC(t *testing.T) {
	now := time.Date(2026, 5, 4, 15, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	r, err := contest.NewContestRegistration("reg-1", "contest-1", "user-1", now)

	if err != nil {
		t.Fatalf("NewContestRegistration: %v", err)
	}
	if r.ID() != "reg-1" || r.ContestID() != "contest-1" || r.UserID() != "user-1" {
		t.Errorf("fields = %s/%s/%s, want reg-1/contest-1/user-1", r.ID(), r.ContestID(), r.UserID())
	}
	want := time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC)
	if !r.RegisteredAt().Equal(want) || r.RegisteredAt().Location() != time.UTC {
		t.Errorf("RegisteredAt = %v, want %v in UTC", r.RegisteredAt(), want)
	}
}

func TestNewContestRegistration_MissingData_IsAnInternalError(t *testing.T) {
	tests := []struct {
		name                  string
		id, contestID, userID string
		now                   time.Time
	}{
		{"an empty id", "", "contest-1", "user-1", fixedNow},
		{"an empty contest id", "reg-1", "", "user-1", fixedNow},
		{"an empty user id", "reg-1", "contest-1", "", fixedNow},
		{"a zero instant", "reg-1", "contest-1", "user-1", time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := contest.NewContestRegistration(tt.id, tt.contestID, tt.userID, tt.now)

			assertInternalError(t, tt.name, err)
			if r != nil {
				t.Errorf("expected no registration, got %+v", r)
			}
		})
	}
}

func TestRestoreContestRegistration_KeepsTheValuesAndNormalisesTheInstant(t *testing.T) {
	registeredAt := time.Date(2026, 5, 4, 15, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	r := contest.RestoreContestRegistration("", "contest-1", "user-1", registeredAt)

	if r.ID() != "" {
		t.Errorf("ID = %q, want the empty id kept as stored", r.ID())
	}
	if want := time.Date(2026, 5, 4, 13, 0, 0, 0, time.UTC); !r.RegisteredAt().Equal(want) || r.RegisteredAt().Location() != time.UTC {
		t.Errorf("RegisteredAt = %v, want %v in UTC", r.RegisteredAt(), want)
	}
}
