package contest

import (
	"context"
	"log/slog"
)

// ContestParticipantProvider is a no-op stub. Replace with a real implementation when participant registration is supported.
type ContestParticipantProvider struct{}

func NewContestParticipantProvider() *ContestParticipantProvider {
	return &ContestParticipantProvider{}
}

func (p *ContestParticipantProvider) IsRegistered(ctx context.Context, _, _ string) (bool, error) {
	slog.WarnContext(ctx, "ContestParticipantProvider is a no-op stub; IsRegistered always returns false")
	return false, nil
}

func (p *ContestParticipantProvider) CountParticipants(ctx context.Context, _ string) (int, error) {
	slog.WarnContext(ctx, "ContestParticipantProvider is a no-op stub; CountParticipants always returns 0")
	return 0, nil
}

func (p *ContestParticipantProvider) CountParticipantsBulk(ctx context.Context, contestIDs []string) (map[string]int, error) {
	slog.WarnContext(ctx, "ContestParticipantProvider is a no-op stub; CountParticipantsBulk always returns 0")
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	return result, nil
}

func (p *ContestParticipantProvider) IsRegisteredBulk(ctx context.Context, contestIDs []string, _ string) (map[string]bool, error) {
	slog.WarnContext(ctx, "ContestParticipantProvider is a no-op stub; IsRegisteredBulk always returns false")
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = false
	}
	return result, nil
}
