package contest

type StandingsPhase string

const (
	StandingsPhaseCompetition     StandingsPhase = "competition"
	StandingsPhasePostcompetition StandingsPhase = "postcompetition"
	StandingsPhaseAll             StandingsPhase = "all"
)

func (p StandingsPhase) withCompetition() bool {
	return p != StandingsPhasePostcompetition
}

func (p StandingsPhase) withPostcompetition() bool {
	return p == StandingsPhasePostcompetition || p == StandingsPhaseAll || p == ""
}
