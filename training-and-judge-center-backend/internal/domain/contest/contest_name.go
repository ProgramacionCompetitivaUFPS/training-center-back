package contest

import (
	"fmt"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const MaxContestNameLength = 200

type ContestName struct {
	value string
}

func NewContestName(s string) (ContestName, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return ContestName{}, apperror.NewBadRequest(ErrCodeInvalidContestName, "contest name cannot be empty")
	}
	if len([]rune(trimmed)) > MaxContestNameLength {
		return ContestName{}, apperror.NewBadRequest(ErrCodeInvalidContestName, fmt.Sprintf("contest name cannot exceed %d characters", MaxContestNameLength))
	}
	return ContestName{value: trimmed}, nil
}

func RestoreContestName(s string) ContestName {
	return ContestName{value: s}
}

func (n ContestName) Value() string  { return n.value }
func (n ContestName) String() string { return n.value }
