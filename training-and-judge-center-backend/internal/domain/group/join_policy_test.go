package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewJoinPolicy(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantOK    bool
		wantValue group.JoinPolicy
	}{
		{"invite", "INVITE", true, group.JoinPolicyInvite},
		{"request", "REQUEST", true, group.JoinPolicyRequest},
		{"open", "OPEN", true, group.JoinPolicyOpen},
		{"lowercase rejected", "invite", false, group.JoinPolicy{}},
		{"empty string rejected", "", false, group.JoinPolicy{}},
		{"unknown rejected", "UNKNOWN", false, group.JoinPolicy{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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
		})
	}
}
