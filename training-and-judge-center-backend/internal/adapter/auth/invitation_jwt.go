package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	invitationTokenDuration = 72 * time.Hour
	invitationIssuer        = "group-invite"
)

type invitationJWTClaims struct {
	GroupID   string `json:"group_id"`
	InviterID string `json:"inviter_id"`
	jwt.RegisteredClaims
}

type GroupInvitationJWTService struct {
	secret []byte
}

func NewGroupInvitationJWTService(secret string) *GroupInvitationJWTService {
	return &GroupInvitationJWTService{secret: []byte(secret)}
}

func (s *GroupInvitationJWTService) GenerateInviteToken(groupID, inviterID string) (string, error) {
	claims := invitationJWTClaims{
		GroupID:   groupID,
		InviterID: inviterID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    invitationIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(invitationTokenDuration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", apperror.NewInternal()
	}
	return signed, nil
}

func (s *GroupInvitationJWTService) ValidateInviteToken(tokenString string) (*appGroup.InvitationClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &invitationJWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	}, jwt.WithIssuer(invitationIssuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperror.NewBadRequest(appGroup.ErrCodeExpiredInviteToken, "invitation link has expired")
		}
		return nil, apperror.NewBadRequest(appGroup.ErrCodeInvalidInviteToken, "invalid invitation token")
	}

	claims, ok := token.Claims.(*invitationJWTClaims)
	if !ok || !token.Valid {
		return nil, apperror.NewBadRequest(appGroup.ErrCodeInvalidInviteToken, "invalid invitation token")
	}

	if claims.GroupID == "" {
		return nil, apperror.NewBadRequest(appGroup.ErrCodeInvalidInviteToken, "invalid invitation token")
	}

	return &appGroup.InvitationClaims{
		GroupID: claims.GroupID,
	}, nil
}
