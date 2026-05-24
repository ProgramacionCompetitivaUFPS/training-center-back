package contest

import "context"

type RegistrationRepository interface {
	Save(ctx context.Context, r *ContestRegistration) error
	FindByContestAndUser(ctx context.Context, contestID, userID string) (*ContestRegistration, error)
	Delete(ctx context.Context, contestID, userID string) error
	ListByContest(ctx context.Context, contestID string, page, limit int) ([]*ContestRegistration, int, error)
	ExistsByContestAndUser(ctx context.Context, contestID, userID string) (bool, error)
	CountByContest(ctx context.Context, contestID string) (int, error)
	CountByContestBulk(ctx context.Context, contestIDs []string) (map[string]int, error)
	ExistsByUserBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error)
}
