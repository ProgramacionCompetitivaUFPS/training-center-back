package user

import (
	"fmt"
	"strings"
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

func NewUser(email Email, password Password, name string, nickname Nickname, country, city, institution string) (*User, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if strings.TrimSpace(country) == "" {
		return nil, fmt.Errorf("country is required")
	}
	if strings.TrimSpace(city) == "" {
		return nil, fmt.Errorf("city is required")
	}
	if strings.TrimSpace(institution) == "" {
		return nil, fmt.Errorf("institution is required")
	}

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
	}, nil
}

func (u *User) Update(name *string, nickname *Nickname, institution *string, city *string, country *string) error {
	if name != nil {
		if strings.TrimSpace(*name) == "" {
			return fmt.Errorf("name cannot be empty")
		}
		u.name = *name
	}
	if nickname != nil {
		u.nickname = *nickname
	}
	if institution != nil {
		if strings.TrimSpace(*institution) == "" {
			return fmt.Errorf("institution cannot be empty")
		}
		u.institution = *institution
	}
	if city != nil {
		if strings.TrimSpace(*city) == "" {
			return fmt.Errorf("city cannot be empty")
		}
		u.city = *city
	}
	if country != nil {
		if strings.TrimSpace(*country) == "" {
			return fmt.Errorf("country cannot be empty")
		}
		u.country = *country
	}

	now := time.Now()
	u.updatedAt = &now
	return nil
}

func (u *User) AdminUpdate(name *string, nickname *Nickname, institution *string, city *string, country *string, email *Email, role *Role) error {
	if role != nil && !role.IsValid() {
		return fmt.Errorf("invalid role: %s", *role)
	}
	if role != nil && *role == RoleAdmin {
		return fmt.Errorf("role ADMIN cannot be assigned through standard update")
	}

	if err := u.Update(name, nickname, institution, city, country); err != nil {
		return err
	}

	if email != nil {
		u.email = email
	}
	if role != nil {
		u.role = *role
	}

	now := time.Now()
	u.updatedAt = &now
	return nil
}

func (u *User) UpdatePassword(newPassword Password) {
	u.password = newPassword
	now := time.Now()
	u.updatedAt = &now
}

func (u *User) UpdateEmail(newEmail Email) {
	u.email = &newEmail
	now := time.Now()
	u.updatedAt = &now
}

func (u *User) Deactivate() error {
	if u.status == StatusDeactivated {
		return fmt.Errorf("user is already deactivated")
	}
	anonymousNickname, err := NewNickname("user_anonimo_" + uuid.New().String()[:10])
	if err != nil {
		return fmt.Errorf("generating anonymous nickname: %w", err)
	}
	now := time.Now()
	u.nickname = anonymousNickname
	u.email = nil
	u.status = StatusDeactivated
	u.deactivatedAt = &now
	u.updatedAt = &now
	return nil
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
) (*User, error) {
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
		parsedEmail, err := NewEmail(*emailStr)
		if err != nil {
			return nil, fmt.Errorf("restoring user %s: invalid email: %w", id, err)
		}
		u.email = &parsedEmail
	}

	parsedPassword, err := NewPasswordFromHash(passwordHash)
	if err != nil {
		return nil, fmt.Errorf("restoring user %s: invalid password hash: %w", id, err)
	}
	u.password = parsedPassword

	parsedNickname, err := NewNickname(nicknameStr)
	if err != nil {
		return nil, fmt.Errorf("restoring user %s: invalid nickname: %w", id, err)
	}
	u.nickname = parsedNickname

	parsedRole, err := NewRole(roleStr)
	if err != nil {
		return nil, fmt.Errorf("restoring user %s: invalid role: %w", id, err)
	}
	u.role = parsedRole

	parsedStatus, err := NewStatus(statusStr)
	if err != nil {
		return nil, fmt.Errorf("restoring user %s: invalid status: %w", id, err)
	}
	u.status = parsedStatus

	return u, nil
}
