package login

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/idle-server/common"
)

// MemoryUserRepository 内存用户仓库
type MemoryUserRepository struct {
	users map[string]*common.UserData // username -> user
	byID  map[string]*common.UserData // playerID -> user
	mutex sync.RWMutex
}

// NewMemoryUserRepository 创建内存用户仓库
func NewMemoryUserRepository() common.UserRepository {
	return &MemoryUserRepository{
		users: make(map[string]*common.UserData),
		byID:  make(map[string]*common.UserData),
	}
}

// SaveUser 保存用户
func (r *MemoryUserRepository) SaveUser(user *common.UserData) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.users[user.Username] = user
	r.byID[user.PlayerID] = user
	return nil
}

// GetUser 获取用户
func (r *MemoryUserRepository) GetUser(username string) (*common.UserData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	return user, nil
}

// GetUserByPlayerID 根据PlayerID获取用户
func (r *MemoryUserRepository) GetUserByPlayerID(playerID string) (*common.UserData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	user, exists := r.byID[playerID]
	if !exists {
		return nil, fmt.Errorf("user not found with player ID: %s", playerID)
	}
	return user, nil
}

// UpdateLastLogin 更新最后登录时间
func (r *MemoryUserRepository) UpdateLastLogin(username string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	user, exists := r.users[username]
	if !exists {
		return fmt.Errorf("user not found: %s", username)
	}
	user.LastLogin = time.Now()
	return nil
}

// UserExists 检查用户是否存在
func (r *MemoryUserRepository) UserExists(username string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.users[username]
	return exists
}

// GeneratePlayerID 生成唯一的玩家ID
func GeneratePlayerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
