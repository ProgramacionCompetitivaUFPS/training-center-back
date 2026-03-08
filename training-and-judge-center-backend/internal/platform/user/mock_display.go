package user

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type MockDisplayProvider struct {
	users map[string]*user.Display
}

func NewMockDisplayProvider() *MockDisplayProvider {
	return &MockDisplayProvider{
		users: map[string]*user.Display{
			"aaaaaaaa-0000-0000-0000-000000000001": {Nickname: "admin_demo", Name: "Admin Demo"},
			"aaaaaaaa-0000-0000-0000-000000000002": {Nickname: "coach_john", Name: "John Smith"},
			"aaaaaaaa-0000-0000-0000-000000000003": {Nickname: "coach_mary", Name: "Mary Johnson"},
			"aaaaaaaa-0000-0000-0000-000000000004": {Nickname: "contestant_alex", Name: "Alex Student"},
		},
	}
}

func (m *MockDisplayProvider) GetDisplay(_ context.Context, userID string) (*user.Display, error) {
	if d, ok := m.users[userID]; ok {
		return d, nil
	}
	nick := "user_unknown"
	if len(userID) >= 8 {
		nick = "user_" + userID[:8]
	}
	return &user.Display{Nickname: nick, Name: "Mock User"}, nil
}

func (m *MockDisplayProvider) GetDisplays(_ context.Context, userIDs []string) (map[string]*user.Display, error) {
	out := make(map[string]*user.Display, len(userIDs))
	for _, id := range userIDs {
		d, _ := m.GetDisplay(nil, id)
		out[id] = d
	}
	return out, nil
}
