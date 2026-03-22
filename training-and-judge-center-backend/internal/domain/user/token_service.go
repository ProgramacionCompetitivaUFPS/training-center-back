package user

import "time"

type TokenClaims struct {
	UserID   string
	Email    string
	Role     string
	IssuedAt time.Time
}

type TokenService interface {
	GenerateToken(user *User) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}
