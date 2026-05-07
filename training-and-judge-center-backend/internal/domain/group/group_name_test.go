package group_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/group"
)

func TestNewGroupName_Valid(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantVal string
	}{
		{"algorithms 2025", "Algorithms 2025", "Algorithms 2025"},
		{"trims whitespace", "  trimmed  ", "trimmed"},
		{"single char", "A", "A"},
		{"global contest", "Global Contest", "Global Contest"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, err := group.NewGroupName(tc.input)
			if err != nil {
				t.Errorf("NewGroupName(%q) unexpected error: %v", tc.input, err)
			}
			if n.Value() != tc.wantVal {
				t.Errorf("Value() = %q, want %q", n.Value(), tc.wantVal)
			}
		})
	}
}

func TestNewGroupName_Empty(t *testing.T) {
	for _, input := range []string{"", "   "} {
		_, err := group.NewGroupName(input)
		if err == nil {
			t.Errorf("NewGroupName(%q) expected error, got nil", input)
			continue
		}
		assertValidationField(t, "NewGroupName("+input+")", err, "name")
	}
}

func TestNewGroupName_TooLong(t *testing.T) {
	long := ""
	for i := 0; i < group.MaxGroupNameLength+1; i++ {
		long += "a"
	}
	_, err := group.NewGroupName(long)
	if err == nil {
		t.Fatalf("NewGroupName with %d chars expected error, got nil", group.MaxGroupNameLength+1)
	}
	assertValidationField(t, "NewGroupName(too long)", err, "name")
}

func TestNewGroupName_ReservedName(t *testing.T) {
	for _, reserved := range []string{"global", "Global", "GLOBAL", "  global  "} {
		_, err := group.NewGroupName(reserved)
		if err == nil {
			t.Errorf("NewGroupName(%q) expected reserved-name error, got nil", reserved)
			continue
		}
		assertValidationField(t, "NewGroupName("+reserved+")", err, "name")
	}
}

func TestNewGroupName_MaxLengthValid(t *testing.T) {
	name := ""
	for i := 0; i < group.MaxGroupNameLength; i++ {
		name += "a"
	}
	_, err := group.NewGroupName(name)
	if err != nil {
		t.Errorf("NewGroupName with exactly %d chars should be valid, got: %v", group.MaxGroupNameLength, err)
	}
}

func TestNewGroupName_MultibyteValid(t *testing.T) {
	name := "算法竞赛训练营算法竞"
	n, err := group.NewGroupName(name)
	if err != nil {
		t.Errorf("NewGroupName with 10 CJK chars unexpected error: %v", err)
	}
	if n.Value() != name {
		t.Errorf("Value() = %q, want %q", n.Value(), name)
	}
}

func TestRestoreGroupName(t *testing.T) {
	n := group.RestoreGroupName("My Group")
	if n.Value() != "My Group" {
		t.Errorf("RestoreGroupName Value() = %q, want %q", n.Value(), "My Group")
	}
}
