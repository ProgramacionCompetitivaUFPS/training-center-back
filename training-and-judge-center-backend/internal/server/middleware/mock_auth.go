package middleware

import (
	"context"
	"net/http"
	"os"

	"github.com/training-judge-center/backend/internal/domain/user"
)

const currentUserKey contextKey = "currentUser"

type MockUser struct {
	ID   string
	Role user.Role
}

var mockUsers = map[string]MockUser{
	"admin":      {"aaaaaaaa-0000-0000-0000-000000000001", user.RoleAdmin},
	"coach_john": {"aaaaaaaa-0000-0000-0000-000000000002", user.RoleCoach},
	"coach_mary": {"aaaaaaaa-0000-0000-0000-000000000003", user.RoleCoach},
	"contestant": {"aaaaaaaa-0000-0000-0000-000000000004", user.RoleContestant},
}

const mockUserHeader = "X-Mock-User"

func MockAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("MOCK_AUTH") != "1" {
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get(mockUserHeader)
		mu, ok := mockUsers[key]
		if !ok {
			// No user set → handlers will return 401 Unauthorized
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), currentUserKey, user.CurrentUser{
			ID:   mu.ID,
			Role: mu.Role,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetCurrentUser(ctx context.Context) *user.CurrentUser {
	u, _ := ctx.Value(currentUserKey).(user.CurrentUser)
	if u.ID == "" {
		return nil
	}
	return &u
}
