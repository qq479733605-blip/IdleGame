package user

import "context"

// Repository defines storage behavior for users.
type Repository interface {
	Save(ctx context.Context, user *User) error
	FindByUsername(ctx context.Context, username string) (*User, error)
}

// ErrNotFound is returned when a user does not exist.
type ErrNotFound struct{}

func (ErrNotFound) Error() string { return "user not found" }
