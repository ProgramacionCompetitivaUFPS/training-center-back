package user_test

import (
	"reflect"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewDeactivationStatus(t *testing.T) {
	tests := []struct {
		raw     string
		wantErr bool
	}{
		{"PENDING", false},
		{"CONFIRMED", false},
		{"EXPIRED", false},
		{"BLOCKED", false},
		{"", true},
		{"pending", true},
		{"CANCELLED", true},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := user.NewDeactivationStatus(tt.raw)

			if tt.wantErr {
				if fields := detailFields(t, err); !reflect.DeepEqual(fields, []string{"status"}) {
					t.Errorf("fields = %v, want [status]", fields)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewDeactivationStatus(%q): %v", tt.raw, err)
			}
			if got.String() != tt.raw {
				t.Errorf("String() = %q, want %q", got.String(), tt.raw)
			}
		})
	}
}

func TestRestoreDeactivationStatus_KeepsValueWithoutValidating(t *testing.T) {
	if got := user.RestoreDeactivationStatus("LEGACY").String(); got != "LEGACY" {
		t.Errorf("String() = %q, want LEGACY", got)
	}
}
