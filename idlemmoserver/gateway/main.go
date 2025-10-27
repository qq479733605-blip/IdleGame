package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/idle-server/gateway/internal/gate"
)

func main() {
	log.Println("Starting Gateway Service (New Architecture)...")

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建网关服务
	gatewayService := gate.NewService()

	// 启动服务
	if err := gatewayService.Start(ctx); err != nil {
		log.Fatalf("Failed to start gateway service: %v", err)
	}

	// 设置 Gin 服务器模式
	gin.SetMode(gin.ReleaseMode)

	// 获取 Gin 路由器
	router := gatewayService.GetHTTPHandler()

	// 设置HTTP服务器 - 临时使用端口8006避免冲突
	port := 8005
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// 启动HTTP服务器
	go func() {
		log.Printf("Gateway service listening on port %d", port)
		log.Printf("WebSocket endpoint: ws://localhost:%d/api/ws", port)
		log.Printf("Health check: http://localhost:%d/health", port)
		log.Printf("Debug info: http://localhost:%d/debug", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Gateway Service...")

	// 优雅关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during HTTP server shutdown: %v", err)
	}

	// 关闭网关服务
	if err := gatewayService.Stop(shutdownCtx); err != nil {
		log.Printf("Error during gateway service shutdown: %v", err)
	}

	log.Println("Gateway Service stopped")
}
