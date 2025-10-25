package userjson

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	coreuser "idlemmoserver/internal/core/user"
)

// Repository implements the coreuser.Repository interface using a JSON file.
type Repository struct {
	mu   sync.Mutex
	path string
}

// New creates a JSON repository at the provided path.
func New(path string) *Repository {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)
	return &Repository{path: path}
}

func (r *Repository) Save(ctx context.Context, user *coreuser.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.load()
	if err != nil {
		return err
	}

	users[user.Username] = user
	return r.persist(users)
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (*coreuser.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users, err := r.load()
	if err != nil {
		return nil, err
	}

	if u, ok := users[username]; ok {
		copy := *u
		return &copy, nil
	}
	return nil, coreuser.ErrNotFound{}
}

func (r *Repository) load() (map[string]*coreuser.User, error) {
	users := make(map[string]*coreuser.User)
	b, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return users, nil
	}
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return users, nil
	}
	if err := json.Unmarshal(b, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *Repository) persist(users map[string]*coreuser.User) error {
	b, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, b, 0o644)
}
