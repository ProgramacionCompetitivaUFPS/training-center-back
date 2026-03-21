package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID            string
	Email         *Email
	Password      Password
	Name          string
	Nickname      Nickname
	Country       string
	City          string
	Institution   string
	Role          Role
	Status        Status
	CreatedAt     time.Time
	UpdatedAt     *time.Time
	DeactivatedAt *time.Time
}

func NewUser(email Email, password Password, name string, nickname Nickname, country, city, institution string) *User {
	return &User{
		ID:          uuid.New().String(),
		Email:       &email,
		Password:    password,
		Name:        name,
		Nickname:    nickname,
		Country:     country,
		City:        city,
		Institution: institution,
		Role:        RoleContestant,
		Status:      StatusActive,
		CreatedAt:   time.Now(),
	}
}

func (u *User) Deactivate() {
	anonymousNickname, _ := NewNickname("user_anonimo_" + uuid.New().String()[:10])
	now := time.Now()
	u.Nickname = anonymousNickname
	u.Email = nil
	u.Status = StatusDeactivated
	u.DeactivatedAt = &now
	u.UpdatedAt = &now
}
