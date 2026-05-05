package contest

// ContestProblem links a problem to a contest with a display order.
// Identity within the aggregate is (problemID, order) — the id field
// is an opaque storage key used only by the persistence adapter.
type ContestProblem struct {
	id        string
	problemID string
	order     int
}

func newContestProblem(id, problemID string, order int) ContestProblem {
	return ContestProblem{id: id, problemID: problemID, order: order}
}

func RestoreContestProblem(id, problemID string, order int) ContestProblem {
	return ContestProblem{id: id, problemID: problemID, order: order}
}

func (cp ContestProblem) ID() string        { return cp.id }
func (cp ContestProblem) ProblemID() string { return cp.problemID }
func (cp ContestProblem) Order() int        { return cp.order }

// Letter returns the display alias for this problem slot (1→A, 2→B, …).
func (cp ContestProblem) Letter() string {
	return string(rune('A' + cp.order - 1))
}
