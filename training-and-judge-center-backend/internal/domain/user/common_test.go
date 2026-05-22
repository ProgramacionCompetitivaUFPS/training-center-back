package user_test

import (
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func makeTestUser(t *testing.T) *user.User {
	t.Helper()
	email, _ := user.NewEmail("test@example.com")
	password, _ := user.NewPassword("SecurePass1!")
	nickname, _ := user.NewNickname("user-test")
	u, err := user.NewUser("user-id", testNow, email, password, "John Doe", nickname, "Colombia", "Bogotá", "UFPS")
	if err != nil {
		t.Fatalf("makeTestUser: %v", err)
	}
	return u
}
