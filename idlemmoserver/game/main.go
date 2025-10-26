package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/idle-server/game/internal/game"
	"github.com/nats-io/nats.go"
)

func main() {
	log.Println("Starting Game Service...")

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 连接NATS
	nc, err := nats.Connect(common.NATSURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// 创建Actor系统
	system := actor.NewActorSystem()
	defer system.Shutdown()

	// 创建游戏管理器Actor
	gameManagerProps := actor.PropsFromProducer(func() actor.Actor {
		return game.NewGameManagerActor(system, nc)
	})
	gameManagerPID := system.Root.Spawn(gameManagerProps)

	// 初始化序列配置
	err = game.InitializeConfigs()
	if err != nil {
		log.Printf("Warning: Failed to initialize configs: %v", err)
	}

	log.Printf("Game service started successfully")

	// 启动心跳
	go startHeartbeat(ctx, gameManagerPID)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Game Service...")
	cancel()

	// 停止游戏管理器
	system.Root.Stop(gameManagerPID)

	log.Println("Game Service stopped")
}

// startHeartbeat 启动心跳
func startHeartbeat(ctx context.Context, gameManagerPID *actor.PID) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 创建一个简单的actor来处理心跳
	system := actor.NewActorSystem()
	root := system.Root

	heartbeatProps := actor.PropsFromProducer(func() actor.Actor {
		return &HeartbeatActor{targetPID: gameManagerPID}
	})
	heartbeatPID := root.Spawn(heartbeatProps)

	for {
		select {
		case <-ctx.Done():
			root.Stop(heartbeatPID)
			return
		case <-ticker.C:
			// 发送心跳消息
			root.Send(heartbeatPID, &struct{}{})
		}
	}
}

// HeartbeatActor 心跳Actor
type HeartbeatActor struct {
	targetPID *actor.PID
}

func (h *HeartbeatActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case struct{}:
		// 转发心跳到游戏管理器
		ctx.Send(h.targetPID, &struct{}{})
	}
}
