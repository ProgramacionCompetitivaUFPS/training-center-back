package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

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

func TestVisibility_StringAndRestore(t *testing.T) {
	if group.VisibilityVisible.String() != "VISIBLE" || group.VisibilityNotVisible.String() != "NOT_VISIBLE" {
		t.Errorf("String() = %s/%s, want VISIBLE/NOT_VISIBLE", group.VisibilityVisible, group.VisibilityNotVisible)
	}
	if got := group.RestoreVisibility("LEGACY").String(); got != "LEGACY" {
		t.Errorf("RestoreVisibility().String() = %q, want LEGACY: no validation on restore", got)
	}
}
