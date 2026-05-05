package contest

type Status string

const (
	StatusScheduled Status = "SCHEDULED"
	StatusActive    Status = "ACTIVE"
	StatusFinished  Status = "FINISHED"
)

func (s Status) String() string { return string(s) }
