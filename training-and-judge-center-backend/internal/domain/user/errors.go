package user

import "errors"

const (
	ErrCodeCannotSelfDeactivate  = "CANNOT_SELF_DEACTIVATE"
	ErrCodeCannotDeactivateAdmin = "CANNOT_DEACTIVATE_ADMIN"
)

var (
	ErrNicknameConflict = errors.New("nickname already in use")
	ErrEmailConflict    = errors.New("email already in use")
)

