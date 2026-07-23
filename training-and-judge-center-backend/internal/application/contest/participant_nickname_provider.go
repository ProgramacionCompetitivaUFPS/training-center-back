package contest

import "context"

type ParticipantNicknameProvider interface {
	GetNicknamesByIDs(ctx context.Context, userIDs []string) (map[string]string, error)
}
