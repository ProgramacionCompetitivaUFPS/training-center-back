package group

import (
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const MaxGroupNameLength = 300

type GroupName struct {
	value string
}

func NewGroupName(s string) (GroupName, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return GroupName{}, apperror.NewBadRequest(ErrCodeInvalidName, "group name cannot be empty")
	}
	if len([]rune(trimmed)) > MaxGroupNameLength {
		return GroupName{}, apperror.NewBadRequest(ErrCodeInvalidName, "group name cannot exceed 100 characters")
	}
	if strings.EqualFold(trimmed, "global") {
		return GroupName{}, apperror.NewBadRequest(ErrCodeReservedName, "the group name 'global' is reserved")
	}
	return GroupName{value: trimmed}, nil
}

func RestoreGroupName(s string) GroupName {
	return GroupName{value: s}
}

func (n GroupName) Value() string  { return n.value }
func (n GroupName) String() string { return n.value }
