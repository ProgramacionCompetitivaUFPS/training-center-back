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
	var materialsCount, submissionsCount, membersCount, activeContestsCount int

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
		if err := q.QueryRow(gCtx, `SELECT COUNT(*) FROM contests WHERE group_id = $1 AND start_time <= NOW() AND end_time >= NOW()`, groupID).Scan(&activeContestsCount); err != nil {
			slog.ErrorContext(gCtx, "DeletionProvider: failed to count active contests", "error", err, "group_id", groupID)
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
		ContestIDs:          contestIDs,
		ActiveContestsCount: activeContestsCount,
		MaterialsCount:      materialsCount,
		SubmissionsCount:    submissionsCount,
		MembersCount:        membersCount,
	}, nil
}

// ListSoleLeadGroups returns the groups where the user is the only active lead: if the user is
// deactivated, nobody but an Admin can manage them any more. A deactivated user keeps their
// membership, so another lead who is deactivated does not count as someone who can still manage it.
func (p *DeletionProvider) ListSoleLeadGroups(ctx context.Context, userID string) ([]appGroup.SoleLeadGroup, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)
	rows, err := q.Query(ctx, `
		SELECT g.id, g.name
		FROM group_members m
		JOIN groups g ON g.id = m.group_id
		WHERE m.user_id = $1::uuid AND m.member_role = 'LEAD'
		  AND (SELECT COUNT(*)
		       FROM group_members l
		       JOIN users lu ON lu.id = l.user_id
		       WHERE l.group_id = m.group_id AND l.member_role = 'LEAD' AND lu.status = 'ACTIVE') = 1
		ORDER BY g.name`, userID)
	if err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: failed to list sole-lead groups", "error", err, "user_id", userID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := []appGroup.SoleLeadGroup{}
	for rows.Next() {
		var g appGroup.SoleLeadGroup
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			slog.ErrorContext(ctx, "DeletionProvider: failed to scan sole-lead group", "error", err)
			return nil, apperror.NewInternal()
		}
		result = append(result, g)
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "DeletionProvider: sole-lead groups rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
