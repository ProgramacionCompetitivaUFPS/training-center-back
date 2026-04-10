package user

import "errors"

var (
	ErrCannotSelfDeactivate  = errors.New("cannot self-deactivate")
	ErrCannotDeactivateAdmin = errors.New("cannot deactivate admin")
	ErrNicknameConflict      = errors.New("nickname already in use")
	ErrEmailConflict         = errors.New("email already in use")
)

