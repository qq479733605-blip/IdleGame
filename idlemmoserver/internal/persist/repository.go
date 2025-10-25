package persist

import "idlemmoserver/internal/domain"

type PlayerData struct {
	PlayerID          string
	SeqLevels         map[string]int
	Inventory         map[string]int64
	Exp               int64
	Equipment         map[string]domain.EquipmentState
	OfflineLimitHours int64
}

type Repository interface {
	SavePlayer(data *PlayerData) error
	LoadPlayer(playerID string) (*PlayerData, error)
}
