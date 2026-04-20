package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func assertErrCode(t *testing.T, label string, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected error with code %q, got nil", label, wantCode)
		return
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("%s: expected *apperror.AppError, got %T", label, err)
	}
	if appErr.Code != wantCode {
		t.Errorf("%s: Code = %q, want %q", label, appErr.Code, wantCode)
	}
}

func TestNewJoinPolicy(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantValue group.JoinPolicy
	}{
		{"INVITE", true, group.JoinPolicyInvite},
		{"REQUEST", true, group.JoinPolicyRequest},
		{"OPEN", true, group.JoinPolicyOpen},
		{"invite", false, ""},
		{"", false, ""},
		{"UNKNOWN", false, ""},
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
			assertErrCode(t, "NewJoinPolicy("+tc.input+")", err, group.ErrCodeInvalidJoinPolicy)
		}
	}
}

func TestNewVisibility(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantValue group.Visibility
	}{
		{"VISIBLE", true, group.VisibilityVisible},
		{"NOT_VISIBLE", true, group.VisibilityNotVisible},
		{"visible", false, ""},
		{"", false, ""},
		{"HIDDEN", false, ""},
	}
	for _, tc := range cases {
		got, err := group.NewVisibility(tc.input)
		if tc.wantOK {
			if err != nil {
				t.Errorf("NewVisibility(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.wantValue {
				t.Errorf("NewVisibility(%q) = %q, want %q", tc.input, got, tc.wantValue)
			}
		} else {
			assertErrCode(t, "NewVisibility("+tc.input+")", err, group.ErrCodeInvalidVisibility)
		}
	}
}

func TestNewMemberRole(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantValue group.MemberRole
	}{
		{"LEAD", true, group.MemberRoleLead},
		{"MEMBER", true, group.MemberRoleMember},
		{"lead", false, ""},
		{"", false, ""},
		{"ADMIN", false, ""},
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
			assertErrCode(t, "NewMemberRole("+tc.input+")", err, group.ErrCodeInvalidMemberRole)
		}
	}
}
