package user

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	ExistsByEmail(ctx context.Context, email Email) (bool, error)
	ExistsByNickname(ctx context.Context, nickname Nickname) (bool, error)
	FindByEmail(ctx context.Context, email Email) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByNickname(ctx context.Context, nickname Nickname) (*User, error)
	FindAll(ctx context.Context, filter UserFilter) ([]*User, int, error)
}
