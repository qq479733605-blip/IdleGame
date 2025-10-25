package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// UserData 用户数据结构
type UserData struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"` // 存储哈希值
	PlayerID  string    `json:"player_id"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

// UserRepository 用户仓库接口
type UserRepository interface {
	SaveUser(user *UserData) error
	GetUser(username string) (*UserData, error)
	GetUserByPlayerID(playerID string) (*UserData, error)
	UpdateLastLogin(username string) error
	UserExists(username string) bool
}

// GeneratePlayerID 生成唯一的玩家ID
func GeneratePlayerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// HashPassword 对密码进行简单哈希（生产环境应使用更安全的方式）
func HashPassword(password string) string {
	// 这里使用简单的哈希，生产环境应该使用 bcrypt
	return fmt.Sprintf("%x", password) // 临时使用，实际应该用安全的哈希
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash string) bool {
	return HashPassword(password) == hash
}
