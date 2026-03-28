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

func (u *User) Update(name *string, nickname *Nickname, institution *string, email *Email, role *Role) {
	if name != nil {
		u.Name = *name
	}
	if nickname != nil {
		u.Nickname = *nickname
	}
	if institution != nil {
		u.Institution = *institution
	}
	if email != nil {
		u.Email = email
	}
	if role != nil {
		u.Role = *role
	}
	
	now := time.Now()
	u.UpdatedAt = &now
}

func (u *User) UpdatePassword(newPassword Password) {
	u.Password = newPassword
	now := time.Now()
	u.UpdatedAt = &now
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

func RestoreUser(
	id string,
	emailStr *string,
	passwordHash string,
	name string,
	nicknameStr string,
	country string,
	city string,
	institution string,
	roleStr string,
	statusStr string,
	createdAt time.Time,
	updatedAt *time.Time,
	deactivatedAt *time.Time,
) *User {
	u := &User{
		ID:            id,
		Name:          name,
		Country:       country,
		City:          city,
		Institution:   institution,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		DeactivatedAt: deactivatedAt,
	}

	if emailStr != nil {
		parsedEmail, _ := NewEmail(*emailStr)
		u.Email = &parsedEmail
	}

	u.Password = NewPasswordFromHash(passwordHash)

	parsedNickname, _ := NewNickname(nicknameStr)
	u.Nickname = parsedNickname

	parsedRole, _ := NewRole(roleStr)
	u.Role = parsedRole

	parsedStatus, _ := NewStatus(statusStr)
	u.Status = parsedStatus

	return u
}
