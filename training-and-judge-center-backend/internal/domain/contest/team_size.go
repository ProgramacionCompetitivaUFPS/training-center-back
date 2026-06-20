package contest

import "github.com/training-judge-center/backend/pkg/apperror"

type TeamSize struct {
	min int
	max int
}

func NewTeamSize(min, max int) (TeamSize, error) {
	var errs []apperror.FieldError
	if min < 1 {
		errs = append(errs, apperror.FieldError{Field: "teamSizeMin", Message: "must be at least 1"})
	}
	if max < min {
		errs = append(errs, apperror.FieldError{Field: "teamSizeMax", Message: "must be greater than or equal to teamSizeMin"})
	}
	if len(errs) > 0 {
		return TeamSize{}, apperror.NewValidation(errs)
	}
	return TeamSize{min: min, max: max}, nil
}

func RestoreTeamSize(min, max int) TeamSize { return TeamSize{min: min, max: max} }

func (t TeamSize) Min() int { return t.min }
func (t TeamSize) Max() int { return t.max }

func (t TeamSize) IsValidCount(n int) bool { return n >= t.min && n <= t.max }
