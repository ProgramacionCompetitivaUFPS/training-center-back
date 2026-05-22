package contest

// ContestProblem links a problem to a contest with a display order.
// The id field is an opaque storage key; domain identity within the
// aggregate is problemID — order is a derived display position that
// changes whenever problems are added or removed.
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

// Letter returns the display alias for this problem slot using base-26
// column labels: 1→A, 26→Z, 27→AA, 28→AB, …
func (cp ContestProblem) Letter() string {
	n := cp.order
	result := ""
	for n > 0 {
		n--
		result = string(rune('A'+n%26)) + result
		n /= 26
	}
	return result
}
