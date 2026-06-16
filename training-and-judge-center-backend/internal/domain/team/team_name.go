package team

import (
	"fmt"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
	"golang.org/x/text/unicode/norm"
)

const maxTeamNameLength = 100

type TeamName struct {
	value string
}

func NewTeamName(s string) (TeamName, error) {
	normalized := norm.NFKC.String(strings.TrimSpace(s))
	if normalized == "" {
		return TeamName{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "name", Message: "team name cannot be empty"},
		})
	}
	if len([]rune(normalized)) > maxTeamNameLength {
		return TeamName{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "name", Message: fmt.Sprintf("team name cannot exceed %d characters", maxTeamNameLength)},
		})
	}
	return TeamName{value: normalized}, nil
}

func RestoreTeamName(s string) TeamName {
	return TeamName{value: s}
}

func (n TeamName) Value() string  { return n.value }
func (n TeamName) String() string { return n.value }
