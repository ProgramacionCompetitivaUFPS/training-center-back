package group

import "context"

// SoleLeadGroup is a group whose only lead is a given user.
type SoleLeadGroup struct {
	ID   string
	Name string
}

// SoleLeadGroupsProvider lists the groups where a user is the only lead.
type SoleLeadGroupsProvider interface {
	ListSoleLeadGroups(ctx context.Context, userID string) ([]SoleLeadGroup, error)
}

// ListSoleLeadGroupsUseCase answers "which groups would be left without a lead if this user went
// away?", so deactivating a user can warn about it. Who may ask is decided by the route.
type ListSoleLeadGroupsUseCase struct {
	provider SoleLeadGroupsProvider
}

func NewListSoleLeadGroupsUseCase(provider SoleLeadGroupsProvider) *ListSoleLeadGroupsUseCase {
	return &ListSoleLeadGroupsUseCase{provider: provider}
}

func (uc *ListSoleLeadGroupsUseCase) Execute(ctx context.Context, userID string) ([]SoleLeadGroup, error) {
	return uc.provider.ListSoleLeadGroups(ctx, userID)
}
