package shared_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

func TestNewGroupID_EmptyIsABadRequest(t *testing.T) {
	id, err := shared.NewGroupID("")

	assertBadRequest(t, "empty group id", err)
	if id.Value() != "" {
		t.Errorf("Value() = %q, want the zero value alongside the error", id.Value())
	}
}

func TestNewGroupID_KeepsTheValue(t *testing.T) {
	id, err := shared.NewGroupID("group-1")

	if err != nil {
		t.Fatalf("NewGroupID: %v", err)
	}
	if id.Value() != "group-1" || id.String() != "group-1" {
		t.Errorf("Value()/String() = %q/%q, want group-1", id.Value(), id.String())
	}
}

func TestRestoreGroupID_KeepsTheValueWithoutValidating(t *testing.T) {
	if got := shared.RestoreGroupID("").Value(); got != "" {
		t.Errorf("Value() = %q, want the empty id kept as stored", got)
	}
	if got := shared.RestoreGroupID("group-1").Value(); got != "group-1" {
		t.Errorf("Value() = %q, want group-1", got)
	}
}
