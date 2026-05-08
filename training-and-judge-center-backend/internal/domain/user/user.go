package user

import (
	"strings"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type User struct {
	id            string
	email         Email
	password      Password
	name          string
	nickname      Nickname
	country       string
	city          string
	institution   string
	role          shared.Role
	status        Status
	createdAt     time.Time
	updatedAt     *time.Time
	deactivatedAt *time.Time
}

// Accessors
func (u *User) ID() string                   { return u.id }
func (u *User) Email() Email                 { return u.email }
func (u *User) Password() Password           { return u.password }
func (u *User) Name() string                 { return u.name }
func (u *User) Nickname() Nickname           { return u.nickname }
func (u *User) Country() string              { return u.country }
func (u *User) City() string                 { return u.city }
func (u *User) Institution() string          { return u.institution }
func (u *User) Role() shared.Role            { return u.role }
func (u *User) Status() Status               { return u.status }
func (u *User) CreatedAt() time.Time         { return u.createdAt }
func (u *User) UpdatedAt() *time.Time        { return u.updatedAt }
func (u *User) DeactivatedAt() *time.Time    { return u.deactivatedAt }

func NewUser(id string, now time.Time, email Email, password Password, name string, nickname Nickname, country, city, institution string) (*User, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	var fieldErrs []apperror.FieldError
	if strings.TrimSpace(name) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "name", Message: "name is required"})
	}
	if strings.TrimSpace(country) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "country", Message: "country is required"})
	}
	if strings.TrimSpace(city) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "city", Message: "city is required"})
	}
	if strings.TrimSpace(institution) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "institution", Message: "institution is required"})
	}
	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	return &User{
		id:          id,
		email:       email,
		password:    password,
		name:        name,
		nickname:    nickname,
		country:     country,
		city:        city,
		institution: institution,
		role:        shared.RoleContestant,
		status:      StatusActive,
		createdAt:   now,
	}, nil
}

func (u *User) Update(name *string, nickname *Nickname, institution *string, city *string, country *string, now time.Time) error {
	var fieldErrs []apperror.FieldError
	if name != nil && strings.TrimSpace(*name) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "name", Message: "name cannot be empty"})
	}
	if institution != nil && strings.TrimSpace(*institution) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "institution", Message: "institution cannot be empty"})
	}
	if city != nil && strings.TrimSpace(*city) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "city", Message: "city cannot be empty"})
	}
	if country != nil && strings.TrimSpace(*country) == "" {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "country", Message: "country cannot be empty"})
	}
	if len(fieldErrs) > 0 {
		return apperror.NewValidation(fieldErrs)
	}

	if name != nil {
		u.name = *name
	}
	if nickname != nil {
		u.nickname = *nickname
	}
	if institution != nil {
		u.institution = *institution
	}
	if city != nil {
		u.city = *city
	}
	if country != nil {
		u.country = *country
	}

	t := now.UTC()
	u.updatedAt = &t
	return nil
}

func (u *User) AdminUpdate(name *string, nickname *Nickname, institution *string, city *string, country *string, email *Email, role *shared.Role, now time.Time) error {
	if role != nil && !role.IsValid() {
		return apperror.NewValidation([]apperror.FieldError{{Field: "role", Message: "invalid role"}})
	}
	if role != nil && *role == shared.RoleAdmin {
		return apperror.NewConflict(ErrCodeCannotAssignAdminRole, "role ADMIN cannot be assigned through standard update")
	}

	if err := u.Update(name, nickname, institution, city, country, now); err != nil {
		return err
	}

	if email != nil {
		u.email = *email
	}
	if role != nil {
		u.role = *role
	}

	return nil
}

func (u *User) UpdatePassword(newPassword Password, now time.Time) error {
	if u.status == StatusDeactivated {
		return apperror.NewConflict(ErrCodeCannotUpdateDeactivated, "cannot update password of a deactivated user")
	}
	u.password = newPassword
	t := now.UTC()
	u.updatedAt = &t
	return nil
}

func (u *User) UpdateEmail(newEmail Email, now time.Time) error {
	if u.status == StatusDeactivated {
		return apperror.NewConflict(ErrCodeCannotUpdateDeactivated, "cannot update email of a deactivated user")
	}
	u.email = newEmail
	t := now.UTC()
	u.updatedAt = &t
	return nil
}

func (u *User) Deactivate(anonymousSuffix string, now time.Time) error {
	if u.status == StatusDeactivated {
		return apperror.NewConflict(ErrCodeAlreadyDeactivated, "user is already deactivated")
	}
	anonymousNickname, err := NewNickname("user_anonimo_" + anonymousSuffix)
	if err != nil {
		return apperror.NewInternal()
	}
	t := now.UTC()
	u.nickname = anonymousNickname
	u.email = Email{}
	u.status = StatusDeactivated
	u.deactivatedAt = &t
	u.updatedAt = &t
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
) *User {
	u := &User{
		id:            id,
		name:          name,
		country:       country,
		city:          city,
		institution:   institution,
		password:      RestorePassword(passwordHash),
		nickname:      RestoreNickname(nicknameStr),
		role:          shared.RestoreRole(roleStr),
		status:        RestoreStatus(statusStr),
		createdAt:     createdAt,
		updatedAt:     updatedAt,
		deactivatedAt: deactivatedAt,
	}

	if emailStr != nil {
		u.email = RestoreEmail(*emailStr)
	}

	return u
}
