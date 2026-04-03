package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type UserID struct {
	value string
}

func NewUserID(value string) (UserID, error) {
	if value == "" {
		return UserID{}, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "user ID cannot be empty")
	}
	return UserID{value: value}, nil
}

func RestoreUserID(value string) UserID {
	return UserID{value: value}
}

func (u UserID) Value() string  { return u.value }
func (u UserID) String() string { return u.value }
