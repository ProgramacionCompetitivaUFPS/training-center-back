package contest

import (
	"context"
)

type ContestParticipantProvider struct {
	reg *RegistrationRepository
}

func NewContestParticipantProvider(reg *RegistrationRepository) *ContestParticipantProvider {
	return &ContestParticipantProvider{reg: reg}
}

func (p *ContestParticipantProvider) IsRegistered(ctx context.Context, contestID, userID string) (bool, error) {
	return p.reg.ExistsByContestAndUser(ctx, contestID, userID)
}

func (p *ContestParticipantProvider) CountParticipants(ctx context.Context, contestID string) (int, error) {
	return p.reg.CountByContest(ctx, contestID)
}

func (p *ContestParticipantProvider) CountParticipantsBulk(ctx context.Context, contestIDs []string) (map[string]int, error) {
	return p.reg.CountByContestBulk(ctx, contestIDs)
}

func (p *ContestParticipantProvider) IsRegisteredBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error) {
	return p.reg.ExistsByUserBulk(ctx, contestIDs, userID)
}
