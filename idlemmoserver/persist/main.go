package main

import (
	"log"

	"github.com/idle-server/common/service"
	"github.com/idle-server/persist/internal/persist"
)

func main() {
	log.Println("Starting Persist Service...")

	// 创建统一的持久化服务
	persistService := persist.NewService()

	// 使用统一的服务运行器
	service.RunService(persistService)
}
