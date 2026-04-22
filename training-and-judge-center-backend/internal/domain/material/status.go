package material

import "github.com/training-judge-center/backend/pkg/apperror"

type Status struct {
	value string
}

func NewStatus(value string) (Status, error) {
	switch value {
	case "DRAFT", "PUBLISHED":
		return Status{value: value}, nil
	default:
		return Status{}, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid material status: "+value)
	}
}

func NewStatusDraft() Status     { return Status{value: "DRAFT"} }
func NewStatusPublished() Status { return Status{value: "PUBLISHED"} }
func RestoreStatus(value string) Status { return Status{value: value} }

func (s Status) String() string    { return s.value }
func (s Status) IsPublished() bool { return s.value == "PUBLISHED" }
func (s Status) IsDraft() bool     { return s.value == "DRAFT" }
