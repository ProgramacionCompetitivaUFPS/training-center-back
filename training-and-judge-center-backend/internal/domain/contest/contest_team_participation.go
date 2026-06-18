package contest

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type ContestTeamParticipation struct {
	id              string
	contestID       string
	teamID          string
	selectedMembers []string
	registeredAt    time.Time
}

func NewContestTeamParticipation(
	id, contestID, teamID string,
	selectedMembers []string,
	now time.Time,
) (*ContestTeamParticipation, error) {
	if id == "" || contestID == "" || teamID == "" {
		return nil, apperror.NewInternal()
	}
	members := make([]string, len(selectedMembers))
	copy(members, selectedMembers)
	return &ContestTeamParticipation{
		id:              id,
		contestID:       contestID,
		teamID:          teamID,
		selectedMembers: members,
		registeredAt:    now.UTC(),
	}, nil
}

func RestoreContestTeamParticipation(
	id, contestID, teamID string,
	selectedMembers []string,
	registeredAt time.Time,
) *ContestTeamParticipation {
	members := make([]string, len(selectedMembers))
	copy(members, selectedMembers)
	return &ContestTeamParticipation{
		id:              id,
		contestID:       contestID,
		teamID:          teamID,
		selectedMembers: members,
		registeredAt:    registeredAt.UTC(),
	}
}

func (p *ContestTeamParticipation) ID() string              { return p.id }
func (p *ContestTeamParticipation) ContestID() string       { return p.contestID }
func (p *ContestTeamParticipation) TeamID() string          { return p.teamID }
func (p *ContestTeamParticipation) SelectedMembers() []string {
	out := make([]string, len(p.selectedMembers))
	copy(out, p.selectedMembers)
	return out
}
func (p *ContestTeamParticipation) RegisteredAt() time.Time { return p.registeredAt }

func (p *ContestTeamParticipation) UpdateSelectedMembers(members []string) {
	updated := make([]string, len(members))
	copy(updated, members)
	p.selectedMembers = updated
}
