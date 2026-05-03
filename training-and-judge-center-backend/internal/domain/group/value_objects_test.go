package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// assertValidationField verifies that err is a VALIDATION_ERROR with at least one FieldError
// whose Field matches wantField.
func assertValidationField(t *testing.T, label string, err error, wantField string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected validation error, got nil", label)
		return
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("%s: expected *apperror.AppError, got %T", label, err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("%s: Code = %q, want %q", label, appErr.Code, apperror.ErrCodeValidationError)
	}
	if len(appErr.Details) == 0 {
		t.Fatalf("%s: expected Details with field %q, got empty", label, wantField)
	}
	if appErr.Details[0].Field != wantField {
		t.Errorf("%s: Details[0].Field = %q, want %q", label, appErr.Details[0].Field, wantField)
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

func TestNewVisibility(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantValue group.Visibility
	}{
		{"VISIBLE", true, group.VisibilityVisible},
		{"NOT_VISIBLE", true, group.VisibilityNotVisible},
		{"visible", false, group.Visibility{}},
		{"", false, group.Visibility{}},
		{"HIDDEN", false, group.Visibility{}},
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
			assertValidationField(t, "NewVisibility("+tc.input+")", err, "visibility")
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
