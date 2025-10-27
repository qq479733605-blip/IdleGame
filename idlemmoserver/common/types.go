package common

import (
	"time"
)

// PlayerData 玩家数据结构 - 简化版本
type PlayerData struct {
	PlayerID     string    `json:"player_id"`
	Username     string    `json:"username"`
	Level        int       `json:"level"`
	Exp          int64     `json:"exp"`
	LastSaveTime time.Time `json:"last_save_time"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserData 用户数据结构
type UserData struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"` // 存储哈希值
	PlayerID  string    `json:"player_id"`
	Level     int       `json:"level"`
	Exp       int64     `json:"exp"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

// PlayerRanking 玩家排行榜结构
type PlayerRanking struct {
	Rank     int    `json:"rank"`
	PlayerID string `json:"player_id"`
	Username string `json:"username"`
	Level    int    `json:"level"`
	Exp      int64  `json:"exp"`
}

// UserRepository 用户仓库接口
type UserRepository interface {
	SaveUser(user *UserData) error
	GetUser(username string) (*UserData, error)
	GetUserByPlayerID(playerID string) (*UserData, error)
	UpdateLastLogin(username string) error
	UserExists(username string) bool
}
