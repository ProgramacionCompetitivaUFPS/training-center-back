package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewMemberRole(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantValue group.MemberRole
	}{
		{"LEAD", true, group.MemberRoleLead},
		{"MEMBER", true, group.MemberRoleMember},
		{"lead", false, group.MemberRole{}},
		{"", false, group.MemberRole{}},
		{"ADMIN", false, group.MemberRole{}},
	}
	for _, tc := range cases {
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
	}
}
