package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewJoinPolicy(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantValue group.JoinPolicy
	}{
		{"INVITE", true, group.JoinPolicyInvite},
		{"REQUEST", true, group.JoinPolicyRequest},
		{"OPEN", true, group.JoinPolicyOpen},
		{"invite", false, group.JoinPolicy{}},
		{"", false, group.JoinPolicy{}},
		{"UNKNOWN", false, group.JoinPolicy{}},
	}
	for _, tc := range cases {
		got, err := group.NewJoinPolicy(tc.input)
		if tc.wantOK {
			if err != nil {
				t.Errorf("NewJoinPolicy(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.wantValue {
				t.Errorf("NewJoinPolicy(%q) = %q, want %q", tc.input, got, tc.wantValue)
			}
		} else {
			assertValidationField(t, "NewJoinPolicy("+tc.input+")", err, "joinPolicy")
		}
	}
}
