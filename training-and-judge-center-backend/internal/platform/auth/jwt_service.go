package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type jwtCustomClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secret     []byte
	expiration time.Duration
}

func NewJWTService(secret string, expirationHours int) *JWTService {
	return &JWTService{
		secret:     []byte(secret),
		expiration: time.Duration(expirationHours) * time.Hour,
	}
}

func (s *JWTService) GenerateToken(u *user.User) (string, error) {
	claims := jwtCustomClaims{
		Email: u.Email().String(),
		Role:  u.Role().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expiration)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) ValidateToken(tokenString string) (*user.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtCustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid token").WithCause(err)
	}

	claims, ok := token.Claims.(*jwtCustomClaims)
	if !ok || !token.Valid {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid token")
	}

	parsedEmail, err := user.NewEmail(claims.Email)
	if err != nil {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid token").WithCause(err)
	}

	parsedRole, err := user.NewRole(claims.Role)
	if err != nil {
		return nil, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid token").WithCause(err)
	}

	return &user.TokenClaims{
		UserID:   claims.Subject,
		Email:    parsedEmail,
		Role:     parsedRole,
		IssuedAt: claims.IssuedAt.Time,
	}, nil
}
