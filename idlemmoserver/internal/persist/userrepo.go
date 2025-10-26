package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"idlemmoserver/internal/common"
)

type JSONUserRepo struct {
	mu   sync.Mutex
	path string
}

func NewJSONUserRepo(userDir string) *JSONUserRepo {
	_ = os.MkdirAll(userDir, 0755)
	return &JSONUserRepo{path: userDir}
}

func (r *JSONUserRepo) filePath(username string) string {
	safe := sanitizeFilename(username)
	return filepath.Join(r.path, fmt.Sprintf("user_%s.json", safe))
}

func sanitizeFilename(name string) string {
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
}

func (r *JSONUserRepo) SaveUser(user *common.UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(user.Username)
	b, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}
	return os.WriteFile(f, b, 0644)
}

func (r *JSONUserRepo) GetUser(username string) (*common.UserData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(username)
	b, err := os.ReadFile(f)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("read user file: %w", err)
	}
	var user common.UserData
	if err := json.Unmarshal(b, &user); err != nil {
		return nil, fmt.Errorf("parse user json: %w", err)
	}
	return &user, nil
}

func (r *JSONUserRepo) GetUserByPlayerID(playerID string) (*common.UserData, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	files, err := os.ReadDir(r.path)
	if err != nil {
		return nil, fmt.Errorf("read user dir: %w", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matched, err := filepath.Match("user_*.json", file.Name())
		if err != nil || !matched {
			continue
		}
		f := filepath.Join(r.path, file.Name())
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var user common.UserData
		if err := json.Unmarshal(b, &user); err != nil {
			continue
		}
		if user.PlayerID == playerID {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found by player_id")
}

func (r *JSONUserRepo) UpdateLastLogin(username string) error {
	user, err := r.GetUser(username)
	if err != nil {
		return err
	}
	user.LastLogin = time.Now()
	return r.SaveUser(user)
}

func (r *JSONUserRepo) UserExists(username string) bool {
	_, err := r.GetUser(username)
	return err == nil
}
