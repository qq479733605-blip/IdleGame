package persist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"idlemmoserver/internal/domain"
)

// JSONUserRepo JSON 用户仓库实现
type JSONUserRepo struct {
	mu   sync.Mutex
	path string
}

// NewJSONUserRepo 创建 JSON 用户仓库
func NewJSONUserRepo(userDir string) *JSONUserRepo {
	_ = os.MkdirAll(userDir, 0755)
	return &JSONUserRepo{path: userDir}
}

// filePath 获取用户文件路径
func (r *JSONUserRepo) filePath(username string) string {
	// 使用用户名作为文件名，确保文件系统安全
	safeUsername := sanitizeFilename(username)
	return filepath.Join(r.path, fmt.Sprintf("user_%s.json", safeUsername))
}

// sanitizeFilename 清理文件名，防止路径遍历攻击
func sanitizeFilename(name string) string {
	// 简单清理，只保留字母数字和下划线
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

// SaveUser 保存用户数据
func (r *JSONUserRepo) SaveUser(user *domain.UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f := r.filePath(user.Username)
	b, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}
	return os.WriteFile(f, b, 0644)
}

// GetUser 获取用户数据
func (r *JSONUserRepo) GetUser(username string) (*domain.UserData, error) {
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

	var user domain.UserData
	if err := json.Unmarshal(b, &user); err != nil {
		return nil, fmt.Errorf("parse user json: %w", err)
	}
	return &user, nil
}

// GetUserByPlayerID 通过 PlayerID 查找用户
func (r *JSONUserRepo) GetUserByPlayerID(playerID string) (*domain.UserData, error) {
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
			continue // 跳过读取失败的文件
		}

		var user domain.UserData
		if err := json.Unmarshal(b, &user); err != nil {
			continue // 跳过解析失败的文件
		}

		if user.PlayerID == playerID {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found by player_id")
}

// UpdateLastLogin 更新最后登录时间
func (r *JSONUserRepo) UpdateLastLogin(username string) error {
	user, err := r.GetUser(username)
	if err != nil {
		return err
	}

	user.LastLogin = time.Now()
	return r.SaveUser(user)
}

// UserExists 检查用户是否存在
func (r *JSONUserRepo) UserExists(username string) bool {
	_, err := r.GetUser(username)
	return err == nil
}
