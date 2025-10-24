package persist

type PlayerData struct {
	PlayerID          string
	SeqLevels         map[string]int
	Inventory         map[string]int64
	Exp               int64
	OfflineLimitHours int64
}

type Repository interface {
	SavePlayer(data *PlayerData) error
	LoadPlayer(playerID string) (*PlayerData, error)
}
