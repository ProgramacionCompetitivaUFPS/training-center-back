package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewJoinMethod(t *testing.T) {
	tests := []struct {
		raw     string
		wantErr bool
	}{
		{"DIRECT_ADD", false},
		{"INVITATION", false},
		{"REQUEST_APPROVED", false},
		{"OPEN_JOIN", false},
		{"", true},
		{"direct_add", true},
		{"UNKNOWN", true},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := group.NewJoinMethod(tt.raw)

			if tt.wantErr {
				assertValidationField(t, tt.raw, err, "joinMethod")
				return
			}
			if err != nil {
				t.Fatalf("NewJoinMethod(%q): %v", tt.raw, err)
			}
			if got.String() != tt.raw {
				t.Errorf("String() = %q, want %q", got.String(), tt.raw)
			}
		})
	}
}

func TestJoinMethod_ConstantsAndRestore(t *testing.T) {
	if group.JoinMethodDirectAdd.String() != "DIRECT_ADD" || group.JoinMethodOpenJoin.String() != "OPEN_JOIN" {
		t.Errorf("constants = %s/%s, want DIRECT_ADD/OPEN_JOIN", group.JoinMethodDirectAdd, group.JoinMethodOpenJoin)
	}
	if got := group.RestoreJoinMethod("LEGACY").String(); got != "LEGACY" {
		t.Errorf("RestoreJoinMethod().String() = %q, want LEGACY: no validation on restore", got)
	}
}
