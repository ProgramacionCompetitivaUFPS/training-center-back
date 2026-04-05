package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	id            string
	email         *Email
	password      Password
	name          string
	nickname      Nickname
	country       string
	city          string
	institution   string
	role          Role
	status        Status
	createdAt     time.Time
	updatedAt     *time.Time
	deactivatedAt *time.Time
}

// Accessors
func (u *User) ID() string                   { return u.id }
func (u *User) Email() *Email                { return u.email }
func (u *User) Password() Password           { return u.password }
func (u *User) Name() string                 { return u.name }
func (u *User) Nickname() Nickname           { return u.nickname }
func (u *User) Country() string              { return u.country }
func (u *User) City() string                 { return u.city }
func (u *User) Institution() string          { return u.institution }
func (u *User) Role() Role                   { return u.role }
func (u *User) Status() Status               { return u.status }
func (u *User) CreatedAt() time.Time         { return u.createdAt }
func (u *User) UpdatedAt() *time.Time        { return u.updatedAt }
func (u *User) DeactivatedAt() *time.Time    { return u.deactivatedAt }

func NewUser(email Email, password Password, name string, nickname Nickname, country, city, institution string) *User {
	return &User{
		id:          uuid.New().String(),
		email:       &email,
		password:    password,
		name:        name,
		nickname:    nickname,
		country:     country,
		city:        city,
		institution: institution,
		role:        RoleContestant,
		status:      StatusActive,
		createdAt:   time.Now(),
	}
}

func (u *User) Update(name *string, nickname *Nickname, institution *string, email *Email, role *Role) {
	if name != nil {
		u.name = *name
	}
	if nickname != nil {
		u.nickname = *nickname
	}
	if institution != nil {
		u.institution = *institution
	}
	if email != nil {
		u.email = email
	}
	if role != nil {
		u.role = *role
	}
	
	now := time.Now()
	u.updatedAt = &now
}

func (u *User) UpdatePassword(newPassword Password) {
	u.password = newPassword
	now := time.Now()
	u.updatedAt = &now
}

func (u *User) Deactivate() {
	anonymousNickname, _ := NewNickname("user_anonimo_" + uuid.New().String()[:10])
	now := time.Now()
	u.nickname = anonymousNickname
	u.email = nil
	u.status = StatusDeactivated
	u.deactivatedAt = &now
	u.updatedAt = &now
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
		id:            id,
		name:          name,
		country:       country,
		city:          city,
		institution:   institution,
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		deactivatedAt: deactivatedAt,
	}

	if emailStr != nil {
		parsedEmail, _ := NewEmail(*emailStr)
		u.email = &parsedEmail
	}

	u.password = NewPasswordFromHash(passwordHash)

	parsedNickname, _ := NewNickname(nicknameStr)
	u.nickname = parsedNickname

	parsedRole, _ := NewRole(roleStr)
	u.role = parsedRole

	parsedStatus, _ := NewStatus(statusStr)
	u.status = parsedStatus

	return u
}
