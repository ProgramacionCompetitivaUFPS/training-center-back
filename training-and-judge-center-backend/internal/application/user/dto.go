package user

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
)

// UserDTO carries user data to the handler layer without exposing the domain object.
// Sensitive fields like the password hash are never included.
type UserDTO struct {
	ID            string
	Email         *string
	Name          string
	Nickname      string
	Country       string
	City          string
	Institution   string
	Role          string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     *time.Time
	DeactivatedAt *time.Time
}

func userToDTO(u *user.User) UserDTO {
	dto := UserDTO{
		ID:            u.ID(),
		Name:          u.Name(),
		Nickname:      u.Nickname().String(),
		Country:       u.Country(),
		City:          u.City(),
		Institution:   u.Institution(),
		Role:          u.Role().String(),
		Status:        u.Status().String(),
		CreatedAt:     u.CreatedAt(),
		UpdatedAt:     u.UpdatedAt(),
		DeactivatedAt: u.DeactivatedAt(),
	}
	if u.Email() != nil {
		s := u.Email().String()
		dto.Email = &s
	}
	return dto
}
