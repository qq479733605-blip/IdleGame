package game

import (
	"log"

	"github.com/idle-server/game/internal/game/domain"
)

// InitializeConfigs 初始化配置
func InitializeConfigs() error {
	// 加载序列配置
	err := domain.LoadConfig("internal/game/domain/config.json")
	if err != nil {
		log.Printf("Warning: Failed to load sequence configs: %v", err)
		// 尝试加载其他配置文件
		err = domain.LoadConfig("internal/game/domain/config_full.json")
		if err != nil {
			log.Printf("Warning: Failed to load full sequence configs: %v", err)
		}
	}

	log.Println("Game configurations initialized")
	return nil
}
