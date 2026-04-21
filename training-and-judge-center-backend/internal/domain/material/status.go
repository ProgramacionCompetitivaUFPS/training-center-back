package material

type Status struct {
	value string
}

func NewStatusDraft() Status {
	return Status{value: "DRAFT"}
}

func NewStatusPublished() Status {
	return Status{value: "PUBLISHED"}
}

func RestoreStatus(value string) Status {
	return Status{value: value}
}

func (s Status) String() string    { return s.value }
func (s Status) IsPublished() bool { return s.value == "PUBLISHED" }
func (s Status) IsDraft() bool     { return s.value == "DRAFT" }
