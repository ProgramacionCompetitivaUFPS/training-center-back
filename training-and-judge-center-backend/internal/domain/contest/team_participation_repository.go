package contest

import "context"

type TeamParticipationRepository interface {
	Save(ctx context.Context, p *ContestTeamParticipation) error
	FindByContestAndTeam(ctx context.Context, contestID, teamID string) (*ContestTeamParticipation, error)
	Update(ctx context.Context, p *ContestTeamParticipation) error
	Delete(ctx context.Context, contestID, teamID string) error
	List(ctx context.Context, contestID string, page, limit int) ([]*ContestTeamParticipation, int, error)
	// AreUsersSelectedInAnyTeam returns, for each given userID, whether that user
	// appears in selected_members of any team registration for the given contest.
	// Optionally excludes one teamID (pass "" to skip exclusion — used on update).
	AreUsersSelectedInAnyTeam(ctx context.Context, contestID string, userIDs []string, excludeTeamID string) (map[string]bool, error)
}
