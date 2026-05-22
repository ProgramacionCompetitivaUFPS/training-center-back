package material

import "context"

type GroupVisibility string

const (
	GroupVisibilityVisible    GroupVisibility = "VISIBLE"
	GroupVisibilityNotVisible GroupVisibility = "NOT_VISIBLE"
)

type GroupVisibilityProvider interface {
	// FindVisibility returns ("VISIBLE"|"NOT_VISIBLE", true, nil) if the group exists,
	// ("", false, nil) if it does not, or ("", false, err) on infrastructure failure.
	FindVisibility(ctx context.Context, groupID string) (GroupVisibility, bool, error)
}
