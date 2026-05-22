package group

import "context"

type PreferencesReader interface {
	HideGlobalGroup(ctx context.Context, userID string) (bool, error)
}
