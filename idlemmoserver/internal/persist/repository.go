package persist

import "idlemmoserver/internal/common"

type PlayerRepository interface {
	SavePlayer(data *common.PlayerSnapshot) error
	LoadPlayer(playerID string) (*common.PlayerSnapshot, error)
}
