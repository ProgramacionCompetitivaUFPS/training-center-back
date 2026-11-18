package contest

import (
	"regexp"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

var problemColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type ProblemColor struct {
	value string
}

func NewProblemColor(s string) (ProblemColor, error) {
	if !problemColorPattern.MatchString(s) {
		return ProblemColor{}, apperror.NewBadRequest(ErrCodeInvalidProblemColor, "problem color must be a hex color in the form #RRGGBB")
	}
	return ProblemColor{value: strings.ToLower(s)}, nil
}

func RestoreProblemColor(s string) ProblemColor {
	return ProblemColor{value: s}
}

func (c ProblemColor) Value() string { return c.value }
