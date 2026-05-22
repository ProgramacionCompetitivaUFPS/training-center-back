package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewMemberRole(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantOK    bool
		wantValue group.MemberRole
	}{
		{"lead", "LEAD", true, group.MemberRoleLead},
		{"member", "MEMBER", true, group.MemberRoleMember},
		{"lowercase lead rejected", "lead", false, group.MemberRole{}},
		{"empty string rejected", "", false, group.MemberRole{}},
		{"admin rejected", "ADMIN", false, group.MemberRole{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := group.NewMemberRole(tc.input)
			if tc.wantOK {
				if err != nil {
					t.Errorf("NewMemberRole(%q) unexpected error: %v", tc.input, err)
				}
				if got != tc.wantValue {
					t.Errorf("NewMemberRole(%q) = %q, want %q", tc.input, got, tc.wantValue)
				}
			} else {
				assertValidationField(t, "NewMemberRole("+tc.input+")", err, "role")
			}
		})
	}
}
