package user

import (
	"context"
	"errors"
	"strings"
	"time"
)

// RegistrationService coordinates user creation logic using the repository abstraction.
type RegistrationService struct {
	repo Repository
}

// NewRegistrationService creates a registration service instance.
func NewRegistrationService(repo Repository) *RegistrationService {
	return &RegistrationService{repo: repo}
}

// ErrUserExists indicates the username has been taken.
var ErrUserExists = errors.New("user already exists")

// Register validates the input and persists a new user using the injected repository.
func (s *RegistrationService) Register(ctx context.Context, username, password string) (*User, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	if existing, err := s.repo.FindByUsername(ctx, username); err == nil && existing != nil {
		return nil, ErrUserExists
	}

	user := &User{
		Username:  username,
		Password:  password,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
