package database

import (
	"time"

	"github.com/idle-server/common"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"size:50;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	PlayerID     string     `gorm:"size:64;uniqueIndex;not null" json:"player_id"`
	CreatedAt    time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	LastLogin    *time.Time `json:"last_login"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`

	// 移除外键约束 - 在应用层通过 PlayerID 关联
}

// Player 玩家模型
type Player struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID      string    `gorm:"size:64;uniqueIndex;not null" json:"player_id"`
	Username      string    `gorm:"size:50;index;not null" json:"username"`
	Level         int       `gorm:"default:1" json:"level"`
	Exp           int64     `gorm:"default:0" json:"exp"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	LastSaveTime  time.Time `gorm:"autoUpdateTime" json:"last_save_time"`
	IsOnline      bool      `gorm:"default:false" json:"is_online"`
	GameData      string    `gorm:"type:json" json:"game_data"`
	TotalPlaytime int64     `gorm:"default:0" json:"total_playtime"`
	LoginCount    int       `gorm:"default:0" json:"login_count"`

	// 移除外键约束 - 在应用层处理关联
	GameProgress []GameProgress `gorm:"-" json:"game_progress,omitempty"`
}

// GameProgress 游戏进度模型
type GameProgress struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	PlayerID      string    `gorm:"size:64;index;not null" json:"player_id"`
	ProgressType  string    `gorm:"size:50;index" json:"progress_type"`
	ProgressKey   string    `gorm:"size:100;not null" json:"progress_key"`
	ProgressValue string    `gorm:"type:json;not null" json:"progress_value"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联 - 移除外键约束，改为应用层关联
	Player *Player `gorm:"-" json:"player,omitempty"`

	// 复合唯一索引
	PlayerIDKey string `gorm:"-" json:"-"`
}

// BeforeCreate GORM钩子 - 创建前
func (gp *GameProgress) BeforeCreate(tx *gorm.DB) error {
	gp.PlayerIDKey = gp.PlayerID + ":" + gp.ProgressType + ":" + gp.ProgressKey
	return nil
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

func (Player) TableName() string {
	return "players"
}

func (GameProgress) TableName() string {
	return "game_progress"
}

// ToUserData 转换为 UserData 结构体
func (u *User) ToUserData() *common.UserData {
	userData := &common.UserData{
		Username:  u.Username,
		Password:  u.PasswordHash, // Include password for authentication
		PlayerID:  u.PlayerID,
		CreatedAt: u.CreatedAt,
	}

	if u.LastLogin != nil {
		userData.LastLogin = *u.LastLogin
	}

	return userData
}

// ToPlayerData 转换为 PlayerData 结构体
func (p *Player) ToPlayerData() *common.PlayerData {
	playerData := &common.PlayerData{
		PlayerID:     p.PlayerID,
		Username:     p.Username,
		LastSaveTime: p.LastSaveTime,
		CreatedAt:    p.CreatedAt,
	}

	return playerData
}

// FromUserData 从 UserData 创建 Player
func FromUserData(userData *common.UserData) *Player {
	return &Player{
		PlayerID:     userData.PlayerID,
		Username:     userData.Username,
		CreatedAt:    userData.CreatedAt,
		LastSaveTime: time.Now(), // 使用当前时间作为初始保存时间
	}
}

// ToCommonPlayerData 转换为公共 PlayerData，包含游戏数据
func (p *Player) ToCommonPlayerData() *common.PlayerData {
	playerData := p.ToPlayerData()

	// 如果需要解析游戏数据中的额外字段，可以在这里处理
	// 例如：Level, Exp 等
	playerData.Level = p.Level
	playerData.Exp = p.Exp

	return playerData
}
