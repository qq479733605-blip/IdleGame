package login

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"idlemmoserver/common"
)

type UserData struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	PlayerID  string    `json:"player_id"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

type UserRepository interface {
	SaveUser(user *UserData) error
	GetUser(username string) (*UserData, error)
	GetUserByPlayerID(playerID string) (*UserData, error)
	UpdateLastLogin(username string) error
	UserExists(username string) bool
}

type jsonUserRepository struct {
	mu       sync.RWMutex
	path     string
	users    map[string]*UserData
	byPlayer map[string]string
}

type repositoryFile struct {
	Users map[string]*UserData `json:"users"`
}

func NewJSONUserRepository(dir string) (UserRepository, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	repo := &jsonUserRepository{
		path:     filepath.Join(dir, "users.json"),
		users:    make(map[string]*UserData),
		byPlayer: make(map[string]string),
	}
	if err := repo.load(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *jsonUserRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(r.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var file repositoryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parse users: %w", err)
	}
	r.users = file.Users
	r.byPlayer = make(map[string]string)
	for username, u := range r.users {
		if u != nil {
			r.byPlayer[u.PlayerID] = username
		}
	}
	return nil
}

func (r *jsonUserRepository) persistLocked() error {
	payload, err := json.MarshalIndent(repositoryFile{Users: r.users}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, payload, 0o644)
}

func (r *jsonUserRepository) SaveUser(user *UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	clone := *user
	r.users[user.Username] = &clone
	r.byPlayer[user.PlayerID] = user.Username
	return r.persistLocked()
}

func (r *jsonUserRepository) GetUser(username string) (*UserData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[username]
	if !ok {
		return nil, fmt.Errorf("user %s not found", username)
	}
	clone := *user
	return &clone, nil
}

func (r *jsonUserRepository) GetUserByPlayerID(playerID string) (*UserData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	username, ok := r.byPlayer[playerID]
	if !ok {
		return nil, fmt.Errorf("player %s not found", playerID)
	}
	user, ok := r.users[username]
	if !ok {
		return nil, fmt.Errorf("user %s not found", username)
	}
	clone := *user
	return &clone, nil
}

func (r *jsonUserRepository) UpdateLastLogin(username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[username]
	if !ok {
		return fmt.Errorf("user %s not found", username)
	}
	user.LastLogin = time.Now()
	return r.persistLocked()
}

func (r *jsonUserRepository) UserExists(username string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.users[username]
	return ok
}

type Service struct {
	repo   UserRepository
	tokens map[string]string
	mu     sync.RWMutex
}

func NewService(repo UserRepository) *Service {
	return &Service{
		repo:   repo,
		tokens: make(map[string]string),
	}
}

func (s *Service) Register(req common.LoginRequest) (*common.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, fmt.Errorf("username and password required")
	}
	if s.repo.UserExists(req.Username) {
		return &common.LoginResponse{Success: false, Message: "username already exists"}, nil
	}

	playerID := generateID()
	user := &UserData{
		Username:  req.Username,
		Password:  hashPassword(req.Password),
		PlayerID:  playerID,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}
	if err := s.repo.SaveUser(user); err != nil {
		return nil, err
	}
	token := s.issueToken(playerID)
	return &common.LoginResponse{
		Success:  true,
		Message:  "registered",
		Token:    token,
		PlayerID: playerID,
	}, nil
}

func (s *Service) Login(req common.LoginRequest) (*common.LoginResponse, error) {
	user, err := s.repo.GetUser(req.Username)
	if err != nil {
		return &common.LoginResponse{Success: false, Message: "invalid credentials"}, nil
	}
	if !verifyPassword(req.Password, user.Password) {
		return &common.LoginResponse{Success: false, Message: "invalid credentials"}, nil
	}
	if err := s.repo.UpdateLastLogin(req.Username); err != nil {
		return nil, err
	}
	token := s.issueToken(user.PlayerID)
	return &common.LoginResponse{
		Success:  true,
		Message:  "ok",
		Token:    token,
		PlayerID: user.PlayerID,
	}, nil
}

func (s *Service) Verify(token string) *common.VerifyResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	playerID, ok := s.tokens[token]
	if !ok {
		return &common.VerifyResponse{Valid: false}
	}
	return &common.VerifyResponse{Valid: true, PlayerID: playerID}
}

func (s *Service) issueToken(playerID string) string {
	token := generateID()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = playerID
	return token
}

func generateID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}

func hashPassword(password string) string {
	return fmt.Sprintf("%x", password)
}

func verifyPassword(password, hash string) bool {
	return hashPassword(password) == hash
}
