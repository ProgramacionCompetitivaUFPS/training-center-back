package group

import "github.com/training-judge-center/backend/pkg/apperror"

type Visibility string

const (
	VisibilityVisible    Visibility = "VISIBLE"
	VisibilityNotVisible Visibility = "NOT_VISIBLE"
)

func NewVisibility(s string) (Visibility, error) {
	switch Visibility(s) {
	case VisibilityVisible, VisibilityNotVisible:
		return Visibility(s), nil
	}
	return "", apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid visibility: "+s)
}

func RestoreVisibility(s string) Visibility { return Visibility(s) }
func (v Visibility) String() string         { return string(v) }
