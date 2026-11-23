package shared_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

func TestNewUserID_EmptyIsABadRequest(t *testing.T) {
	id, err := shared.NewUserID("")

	assertBadRequest(t, "empty user id", err)
	if id.Value() != "" {
		t.Errorf("Value() = %q, want the zero value alongside the error", id.Value())
	}
}

func TestNewUserID_KeepsTheValue(t *testing.T) {
	id, err := shared.NewUserID("user-1")

	if err != nil {
		t.Fatalf("NewUserID: %v", err)
	}
	if id.Value() != "user-1" || id.String() != "user-1" {
		t.Errorf("Value()/String() = %q/%q, want user-1", id.Value(), id.String())
	}
}

func TestRestoreUserID_KeepsTheValueWithoutValidating(t *testing.T) {
	if got := shared.RestoreUserID("").Value(); got != "" {
		t.Errorf("Value() = %q, want the empty id kept as stored", got)
	}
	if got := shared.RestoreUserID("user-1").Value(); got != "user-1" {
		t.Errorf("Value() = %q, want user-1", got)
	}
}
