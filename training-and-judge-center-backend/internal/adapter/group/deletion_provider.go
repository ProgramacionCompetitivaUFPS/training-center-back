package group

import (
	"context"
	"log/slog"

	"golang.org/x/sync/errgroup"
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
	g, gCtx := errgroup.WithContext(ctx)

	var contestIDs []string
	var materialsCount, submissionsCount, membersCount int

	g.Go(func() error {
		rows, err := q.Query(gCtx, `SELECT id FROM contests WHERE group_id = $1`, groupID)
		if err != nil {
			slog.ErrorContext(gCtx, "DeletionProvider: failed to query contest IDs", "error", err, "group_id", groupID)
			return apperror.NewInternal()
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				slog.ErrorContext(gCtx, "DeletionProvider: failed to scan contest ID", "error", err)
				return apperror.NewInternal()
			}
			contestIDs = append(contestIDs, id)
		}
		if err := rows.Err(); err != nil {
			slog.ErrorContext(gCtx, "DeletionProvider: contest IDs rows error", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})

	g.Go(func() error {
		if err := q.QueryRow(gCtx, `SELECT COUNT(*) FROM materials WHERE group_id = $1`, groupID).Scan(&materialsCount); err != nil {
			slog.ErrorContext(gCtx, "DeletionProvider: failed to count materials", "error", err, "group_id", groupID)
			return apperror.NewInternal()
		}
		return nil
	})

	g.Go(func() error {
		err := q.QueryRow(gCtx, `
			SELECT COUNT(*) FROM submissions s
			JOIN contests c ON s.contest_id = c.id
			WHERE c.group_id = $1
		`, groupID).Scan(&submissionsCount)
		if err != nil {
			slog.ErrorContext(gCtx, "DeletionProvider: failed to count submissions", "error", err, "group_id", groupID)
			return apperror.NewInternal()
		}
		return nil
	})

	g.Go(func() error {
		if err := q.QueryRow(gCtx, `SELECT COUNT(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&membersCount); err != nil {
			slog.ErrorContext(gCtx, "DeletionProvider: failed to count members", "error", err, "group_id", groupID)
			return apperror.NewInternal()
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return appGroup.DeletionCounts{}, err
	}
	if contestIDs == nil {
		contestIDs = []string{}
	}

	return appGroup.DeletionCounts{
		ContestIDs:       contestIDs,
		MaterialsCount:   materialsCount,
		SubmissionsCount: submissionsCount,
		MembersCount:     membersCount,
	}, nil
}
