package contest

import "github.com/training-judge-center/backend/pkg/apperror"

type ParticipationMode struct{ value string }

const (
	participationModeIndividual = "INDIVIDUAL"
	participationModeTeam       = "TEAM"
	participationModeMixed      = "MIXED"
)

var (
	ParticipationModeIndividual = ParticipationMode{value: participationModeIndividual}
	ParticipationModeTeam       = ParticipationMode{value: participationModeTeam}
	ParticipationModeMixed      = ParticipationMode{value: participationModeMixed}
)

func NewParticipationMode(raw string) (ParticipationMode, error) {
	switch raw {
	case participationModeIndividual, participationModeTeam, participationModeMixed:
		return ParticipationMode{value: raw}, nil
	default:
		return ParticipationMode{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "participationMode", Message: "must be INDIVIDUAL, TEAM, or MIXED"},
		})
	}
}

func RestoreParticipationMode(raw string) ParticipationMode {
	return ParticipationMode{value: raw}
}

func (m ParticipationMode) String() string { return m.value }

func (m ParticipationMode) AllowsTeams() bool {
	return m.value == participationModeTeam || m.value == participationModeMixed
}
