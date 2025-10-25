package domain

import "time"

// PlayerModel 描述玩家的核心数据结构。
type PlayerModel struct {
	ID               string
	SeqLevels        map[string]int
	Inventory        *Inventory
	Equipment        *EquipmentLoadout
	Exp              int64
	OfflineLimit     time.Duration
	OfflineStart     time.Time
	LastActive       time.Time
	IsOnline         bool
	CurrentSeqID     string
	ActiveSubProject string
}

// NewPlayerModel 创建一个带有默认值的玩家模型。
func NewPlayerModel(playerID string) *PlayerModel {
	return &PlayerModel{
		ID:           playerID,
		SeqLevels:    make(map[string]int),
		Inventory:    NewInventory(200),
		Equipment:    NewEquipmentLoadout(),
		OfflineLimit: 10 * time.Hour,
		IsOnline:     true,
		LastActive:   time.Now(),
	}
}
