package common

import (
	"time"
)

// PlayerData 玩家数据结构
type PlayerData struct {
	PlayerID          string                    `json:"player_id"`
	Username          string                    `json:"username"`
	SeqLevels         map[string]int            `json:"seq_levels"`
	Inventory         *Inventory                `json:"inventory"`
	Exp               int64                     `json:"exp"`
	Equipment         map[string]EquipmentState `json:"equipment"`
	OfflineLimitHours int64                     `json:"offline_limit_hours"`
	LastSaveTime      time.Time                 `json:"last_save_time"`
	CreatedAt         time.Time                 `json:"created_at"`
}

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

// Inventory 库存系统
type Inventory struct {
	Items   map[string]*ItemStack `json:"items"`
	MaxSize int                   `json:"max_size"`
}

// ItemStack 物品堆
type ItemStack struct {
	Item  Item `json:"item"`
	Count int  `json:"count"`
	Slot  int  `json:"slot,omitempty"`
}

// Item 物品
type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DropChance  float64 `json:"drop_chance"`
	Value       int64   `json:"value"`
	IsEquipment bool    `json:"is_equipment"`
}

// Equipment 装备
type Equipment struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	MinLevel    int                `json:"min_level"`
	DropChance  float64            `json:"drop_chance"`
	Attributes  map[string]float64 `json:"attributes"`
	Type        string             `json:"type"`
	Description string             `json:"description"`
}

// EquipmentState 装备状态
type EquipmentState struct {
	Equipment Equipment `json:"equipment"`
	Level     int       `json:"level"`
	Exp       int64     `json:"exp"`
	IsWorn    bool      `json:"is_worn"`
}

// EquipmentBonus 装备加成
type EquipmentBonus struct {
	GainMultiplier  float64 `json:"gain_multiplier"`
	RareChanceBonus float64 `json:"rare_chance_bonus"`
	ExpMultiplier   float64 `json:"exp_multiplier"`
}

// Sequence 修炼序列
type Sequence struct {
	ID         string              `json:"id"`
	Level      int                 `json:"level"`
	Exp        int64               `json:"exp"`
	StartTime  time.Time           `json:"start_time"`
	LastTick   time.Time           `json:"last_tick"`
	SubProject *SequenceSubProject `json:"sub_project,omitempty"`
}

// SequenceSubProject 修炼子项目
type SequenceSubProject struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	UnlockLevel      int     `json:"unlock_level"`
	Description      string  `json:"description"`
	GainMultiplier   float64 `json:"gain_multiplier"`
	RareChanceBonus  float64 `json:"rare_chance_bonus"`
	ExpMultiplier    float64 `json:"exp_multiplier"`
	IntervalModifier float64 `json:"interval_modifier"`
	ExtraDrops       []Item  `json:"extra_drops"`
}

// SequenceConfig 修炼序列配置
type SequenceConfig struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	BaseGain       int64       `json:"base_gain"`
	GrowthFactor   float64     `json:"growth_factor"`
	TickInterval   float64     `json:"tick_interval"`
	LevelUpExp     int64       `json:"level_up_exp"`
	ExpRate        float64     `json:"exp_rate"`
	Drops          []Item      `json:"drops"`
	RareChance     float64     `json:"rare_chance"`
	RareEvents     []RareEvent `json:"rare_events"`
	EquipmentDrops []Equipment `json:"equipment_drops"`
}

// RareEvent 稀有事件
type RareEvent struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Message  string  `json:"message"`
	MultGain float64 `json:"mult_gain"`
}

// TickResult Tick结果
type TickResult struct {
	Gains        int64      `json:"gains"`
	Items        []Item     `json:"items"`
	RareEvt      *RareEvent `json:"rare_evt,omitempty"`
	Level        int        `json:"level"`
	CurExp       int64      `json:"cur_exp"`
	Leveled      bool       `json:"leveled"`
	SubProjectID string     `json:"sub_project_id"`
}
