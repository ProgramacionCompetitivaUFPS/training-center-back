package contest

import "context"

type RegistrationRepository interface {
	Save(ctx context.Context, r *ContestRegistration) error
	ExistsByContestAndUser(ctx context.Context, contestID, userID string) (bool, error)
	CountByContest(ctx context.Context, contestID string) (int, error)
	CountByContestBulk(ctx context.Context, contestIDs []string) (map[string]int, error)
	ExistsByUserBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error)
}
