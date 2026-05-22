package shared

import "github.com/training-judge-center/backend/pkg/apperror"

type GroupID struct {
	value string
}

func NewGroupID(value string) (GroupID, error) {
	if value == "" {
		return GroupID{}, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "group ID cannot be empty")
	}
	return GroupID{value: value}, nil
}

func RestoreGroupID(value string) GroupID {
	return GroupID{value: value}
}

func (g GroupID) Value() string  { return g.value }
func (g GroupID) String() string { return g.value }
