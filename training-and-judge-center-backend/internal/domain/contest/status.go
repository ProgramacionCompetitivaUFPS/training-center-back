package contest

type Status struct{ value string }

var (
	StatusScheduled = Status{value: "SCHEDULED"}
	StatusActive    = Status{value: "ACTIVE"}
	StatusFinished  = Status{value: "FINISHED"}
)

func (s Status) String() string { return s.value }
