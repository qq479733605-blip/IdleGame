package persist

import (
	"github.com/idle-server/common"
)

// PlayerRepository 玩家数据仓库接口
type PlayerRepository interface {
	Save(playerID string, data *common.PlayerData) error
	Load(playerID string) (*common.PlayerData, error)
	Exists(playerID string) bool
	Delete(playerID string) error
}
