package problem

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type UserProvider interface {
	ExistsByID(ctx context.Context, userID string) (bool, error)
	GetDisplay(ctx context.Context, userID string) (*user.Display, error)
	GetDisplays(ctx context.Context, userIDs []string) (map[string]*user.Display, error)
}
type ProblemFileRepository interface {
	UploadFile(ctx context.Context, path string, content []byte) error
	DeleteFile(ctx context.Context, path string) error
}
