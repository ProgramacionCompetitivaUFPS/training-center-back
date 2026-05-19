package contest

import "context"

// ContestParticipantProvider is a stub until the "Register to contest" story is implemented.
type ContestParticipantProvider struct{}

func NewContestParticipantProvider() *ContestParticipantProvider {
	return &ContestParticipantProvider{}
}

func (p *ContestParticipantProvider) IsRegistered(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (p *ContestParticipantProvider) CountParticipants(_ context.Context, _ string) (int, error) {
	return 0, nil
}
