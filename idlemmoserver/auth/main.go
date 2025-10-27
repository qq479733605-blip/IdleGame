package main

import (
	"log"

	"github.com/idle-server/auth/internal/auth"
	"github.com/idle-server/common/service"
)

func main() {
	log.Println("Starting Auth Service...")

	// 创建统一的认证服务
	authService := auth.NewService()

	// 使用统一的服务运行器
	service.RunService(authService)
}
