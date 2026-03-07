package user

type TokenClaims struct {
	UserID string
	Email  string
	Role   string
}

type TokenService interface {
	GenerateToken(user *User) (string, error)
	ValidateToken(tokenString string) (*TokenClaims, error)
}
