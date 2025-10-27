package auth

import (
	"fmt"
	"sync"
	"time"

	"github.com/idle-server/common"
)

// MemoryUserRepository 内存用户仓库实现
type MemoryUserRepository struct {
	users        map[string]*common.UserData // username -> UserData
	playerToUser map[string]string           // playerID -> username
	mu           sync.RWMutex
}

// NewMemoryUserRepository 创建内存用户仓库
func NewMemoryUserRepository() common.UserRepository {
	return &MemoryUserRepository{
		users:        make(map[string]*common.UserData),
		playerToUser: make(map[string]string),
	}
}

// SaveUser 保存用户
func (r *MemoryUserRepository) SaveUser(user *common.UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查用户名是否已存在
	if _, exists := r.users[user.Username]; exists {
		return fmt.Errorf("user %s already exists", user.Username)
	}

	// 保存用户数据
	r.users[user.Username] = user
	r.playerToUser[user.PlayerID] = user.Username

	return nil
}

// GetUser 根据用户名获取用户
func (r *MemoryUserRepository) GetUser(username string) (*common.UserData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, fmt.Errorf("user %s not found", username)
	}

	return user, nil
}

// GetUserByPlayerID 根据PlayerID获取用户
func (r *MemoryUserRepository) GetUserByPlayerID(playerID string) (*common.UserData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	username, exists := r.playerToUser[playerID]
	if !exists {
		return nil, fmt.Errorf("player ID %s not found", playerID)
	}

	user, exists := r.users[username]
	if !exists {
		return nil, fmt.Errorf("user %s not found", username)
	}

	return user, nil
}

// UserExists 检查用户是否存在
func (r *MemoryUserRepository) UserExists(username string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.users[username]
	return exists
}

// UpdateLastLogin 更新最后登录时间
func (r *MemoryUserRepository) UpdateLastLogin(username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[username]
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	user.LastLogin = time.Now()
	return nil
}

// UpdateUser 更新用户信息
func (r *MemoryUserRepository) UpdateUser(user *common.UserData) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Username]; !exists {
		return fmt.Errorf("user %s not found", user.Username)
	}

	r.users[user.Username] = user
	r.playerToUser[user.PlayerID] = user.Username

	return nil
}

// DeleteUser 删除用户
func (r *MemoryUserRepository) DeleteUser(username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[username]
	if !exists {
		return fmt.Errorf("user %s not found", username)
	}

	delete(r.users, username)
	delete(r.playerToUser, user.PlayerID)

	return nil
}

// GetAllUsers 获取所有用户（管理员功能）
func (r *MemoryUserRepository) GetAllUsers() ([]*common.UserData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*common.UserData, 0, len(r.users))
	for _, user := range r.users {
		users = append(users, user)
	}

	return users, nil
}
