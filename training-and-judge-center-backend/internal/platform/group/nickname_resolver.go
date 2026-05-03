package group

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type NicknameResolver struct {
	db *pgxpool.Pool
}

func NewNicknameResolver(db *pgxpool.Pool) *NicknameResolver {
	return &NicknameResolver{db: db}
}

// ResolveByNickname returns (nil, nil) if the nickname does not exist.
func (r *NicknameResolver) ResolveByNickname(ctx context.Context, nickname string) (*appGroup.UserInfo, error) {
	var id, role string
	err := r.db.QueryRow(ctx, `SELECT id, role FROM users WHERE nickname = $1`, nickname).Scan(&id, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "NicknameResolver.ResolveByNickname failed", "error", err, "nickname", nickname)
		return nil, apperror.NewInternal()
	}
	return &appGroup.UserInfo{ID: shared.RestoreUserID(id), Role: role}, nil
}
