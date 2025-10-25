package domain

import "time"

// PlayerModel captures the mutable gameplay state for a player session.
type PlayerModel struct {
	PlayerID         string
	SeqLevels        map[string]int
	Inventory        *Inventory
	Equipment        *EquipmentLoadout
	Exp              int64
	CurrentSeqID     string
	ActiveSubProject string
	OfflineLimit     time.Duration
	OfflineStart     time.Time
	LastActive       time.Time
	IsOnline         bool
}

// NewPlayerModel constructs a PlayerModel with sensible defaults for a fresh player session.
func NewPlayerModel(playerID string) *PlayerModel {
	return &PlayerModel{
		PlayerID:     playerID,
		SeqLevels:    map[string]int{},
		Inventory:    NewInventory(200),
		Equipment:    NewEquipmentLoadout(),
		OfflineLimit: 10 * time.Hour,
		IsOnline:     true,
	}
}
