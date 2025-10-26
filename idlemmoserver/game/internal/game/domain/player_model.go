package domain

import "idlemmoserver/common"

func NewPlayerSnapshot(playerID string) common.PlayerSnapshot {
	return common.PlayerSnapshot{
		PlayerID:          playerID,
		SeqLevels:         make(map[string]int),
		Inventory:         make(map[string]int64),
		Exp:               0,
		OfflineLimitHours: 12,
	}
}
