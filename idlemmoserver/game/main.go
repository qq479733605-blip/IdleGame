package main

import (
	"log"

	"game/internal/game"
	"github.com/idle-server/common/service"
)

func main() {
	log.Println("Starting Game Service...")

	// 创建统一的游戏服务
	gameService := game.NewService()

	// 使用统一的服务运行器
	service.RunService(gameService)
}
