package submission

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type LeadChecker struct {
	db infraPostgres.Querier
}

func NewLeadChecker(db infraPostgres.Querier) *LeadChecker {
	return &LeadChecker{db: db}
}

func (c *LeadChecker) IsLeadOfContestGroup(ctx context.Context, contestID, userID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM contests ct
			JOIN group_members gm ON gm.group_id = ct.group_id
			WHERE ct.id = $1
			  AND gm.user_id = $2
			  AND gm.member_role = 'LEAD'
		)
	`, contestID, userID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to check lead status", "contest_id", contestID, "user_id", userID, "error", err)
		return false, apperror.NewInternal()
	}

	return exists, nil
}

func (c *LeadChecker) IsLeadOfMember(ctx context.Context, leadID, memberID string) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, c.db)

	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM group_members lead
			JOIN group_members member ON member.group_id = lead.group_id
			JOIN groups g ON g.id = lead.group_id
			WHERE lead.user_id = $1
			  AND lead.member_role = 'LEAD'
			  AND member.user_id = $2
			  AND g.is_default = FALSE
		)
	`, leadID, memberID).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "submission: failed to check lead of member", "lead_id", leadID, "member_id", memberID, "error", err)
		return false, apperror.NewInternal()
	}

	return exists, nil
}
