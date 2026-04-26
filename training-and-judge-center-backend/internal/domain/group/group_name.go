package group

import (
	"fmt"
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
		return GroupName{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "name", Message: "group name cannot be empty"},
		})
	}
	if len([]rune(trimmed)) > MaxGroupNameLength {
		return GroupName{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "name", Message: fmt.Sprintf("group name cannot exceed %d characters", MaxGroupNameLength)},
		})
	}
	if strings.EqualFold(trimmed, "global") {
		return GroupName{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "name", Message: "the group name 'global' is reserved"},
		})
	}
	return GroupName{value: trimmed}, nil
}

func RestoreGroupName(s string) GroupName {
	return GroupName{value: s}
}

func (n GroupName) Value() string  { return n.value }
func (n GroupName) String() string { return n.value }
