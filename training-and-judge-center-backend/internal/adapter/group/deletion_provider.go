package group

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeletionProvider struct {
	db infraPostgres.Querier
}

func NewDeletionProvider(db infraPostgres.Querier) *DeletionProvider {
	return &DeletionProvider{db: db}
}

func (p *DeletionProvider) GetDeletionCounts(ctx context.Context, groupID string) (appGroup.DeletionCounts, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	rows, err := q.Query(ctx, `SELECT id FROM contests WHERE group_id = $1`, groupID)
	if err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: failed to query contest IDs", "error", err, "group_id", groupID)
		return appGroup.DeletionCounts{}, apperror.NewInternal()
	}
	defer rows.Close()

	contestIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			slog.ErrorContext(ctx, "DeletionProvider: failed to scan contest ID", "error", err)
			return appGroup.DeletionCounts{}, apperror.NewInternal()
		}
		contestIDs = append(contestIDs, id)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: contest IDs rows error", "error", err)
		return appGroup.DeletionCounts{}, apperror.NewInternal()
	}
	rows.Close()

	var materialsCount int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM materials WHERE group_id = $1`, groupID).Scan(&materialsCount); err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: failed to count materials", "error", err, "group_id", groupID)
		return appGroup.DeletionCounts{}, apperror.NewInternal()
	}

	var submissionsCount int
	if err := q.QueryRow(ctx, `
		SELECT COUNT(*) FROM submissions s
		JOIN contests c ON s.contest_id = c.id
		WHERE c.group_id = $1
	`, groupID).Scan(&submissionsCount); err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: failed to count submissions", "error", err, "group_id", groupID)
		return appGroup.DeletionCounts{}, apperror.NewInternal()
	}

	var membersCount int
	if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&membersCount); err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: failed to count members", "error", err, "group_id", groupID)
		return appGroup.DeletionCounts{}, apperror.NewInternal()
	}

	return appGroup.DeletionCounts{
		ContestIDs:       contestIDs,
		MaterialsCount:   materialsCount,
		SubmissionsCount: submissionsCount,
		MembersCount:     membersCount,
	}, nil
}
