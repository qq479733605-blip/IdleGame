package game

import (
	"encoding/json"
	"log"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// GameManagerActor 游戏管理器Actor
type GameManagerActor struct {
	system  *actor.ActorSystem
	nc      *nats.Conn
	players map[string]*actor.PID // playerID -> PlayerActor PID
}

// NewGameManagerActor 创建游戏管理器Actor
func NewGameManagerActor(system *actor.ActorSystem, nc *nats.Conn) actor.Actor {
	return &GameManagerActor{
		system:  system,
		nc:      nc,
		players: make(map[string]*actor.PID),
	}
}

// Receive 处理消息
func (a *GameManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgRegisterPlayer:
		a.handleRegisterPlayer(ctx, msg)
	case *common.MsgUnregisterPlayer:
		a.handleUnregisterPlayer(ctx, msg)
	case *common.MsgStartSequence:
		a.handleStartSequence(ctx, msg)
	case *common.MsgStopSequence:
		a.handleStopSequence(ctx, msg)
	case *actor.Started:
		a.handleStarted(ctx)
	default:
		log.Printf("GameManagerActor: unknown message type %T", msg)
	}
}

// handleStarted 处理Actor启动
func (a *GameManagerActor) handleStarted(ctx actor.Context) {
	log.Println("GameManagerActor started")

	// 注册NATS处理器
	a.registerNATSHandlers()
}

// handleRegisterPlayer 处理玩家注册
func (a *GameManagerActor) handleRegisterPlayer(ctx actor.Context, msg *common.MsgRegisterPlayer) {
	// 检查玩家是否已注册
	if _, exists := a.players[msg.PlayerID]; exists {
		log.Printf("Player already registered: %s", msg.PlayerID)
		return
	}

	// 创建PlayerActor
	playerProps := actor.PropsFromProducer(func() actor.Actor {
		return NewPlayerActor(msg.PlayerID, a.system, a.nc)
	})
	playerPID := ctx.Spawn(playerProps)

	// 注册玩家
	a.players[msg.PlayerID] = playerPID

	log.Printf("Player registered: %s", msg.PlayerID)
}

// handleUnregisterPlayer 处理玩家注销
func (a *GameManagerActor) handleUnregisterPlayer(ctx actor.Context, msg *common.MsgUnregisterPlayer) {
	// 查找玩家Actor
	if playerPID, exists := a.players[msg.PlayerID]; exists {
		// 停止PlayerActor
		ctx.Stop(playerPID)
		// 从注册表中移除
		delete(a.players, msg.PlayerID)
		log.Printf("Player unregistered: %s", msg.PlayerID)
	} else {
		log.Printf("Player not found for unregistration: %s", msg.PlayerID)
	}
}

// handleStartSequence 处理开始序列
func (a *GameManagerActor) handleStartSequence(ctx actor.Context, msg *common.MsgStartSequence) {
	// 查找玩家Actor
	if playerPID, exists := a.players[msg.PlayerID]; exists {
		ctx.Send(playerPID, msg)
	} else {
		log.Printf("Player not found: %s", msg.PlayerID)
		// 可以选择自动注册玩家或返回错误
	}
}

// handleStopSequence 处理停止序列
func (a *GameManagerActor) handleStopSequence(ctx actor.Context, msg *common.MsgStopSequence) {
	// 查找玩家Actor
	if playerPID, exists := a.players[msg.PlayerID]; exists {
		ctx.Send(playerPID, msg)
	} else {
		log.Printf("Player not found: %s", msg.PlayerID)
	}
}

// registerNATSHandlers 注册NATS处理器
func (a *GameManagerActor) registerNATSHandlers() {
	// 注册玩家注册处理器
	_, err := a.nc.Subscribe(common.GamePlayerRegisterSubject, func(msg *nats.Msg) {
		a.handleNATSPlayerRegister(msg)
	})
	if err != nil {
		log.Printf("Failed to register player registration handler: %v", err)
		return
	}

	// 注册玩家注销处理器
	_, err = a.nc.Subscribe(common.GamePlayerUnregisterSubject, func(msg *nats.Msg) {
		a.handleNATSPlayerUnregister(msg)
	})
	if err != nil {
		log.Printf("Failed to register player unregistration handler: %v", err)
		return
	}

	// 注册开始序列处理器
	_, err = a.nc.Subscribe(common.GameStartSequenceSubject, func(msg *nats.Msg) {
		a.handleNATSStartSequence(msg)
	})
	if err != nil {
		log.Printf("Failed to register start sequence handler: %v", err)
		return
	}

	// 注册停止序列处理器
	_, err = a.nc.Subscribe(common.GameStopSequenceSubject, func(msg *nats.Msg) {
		a.handleNATSStopSequence(msg)
	})
	if err != nil {
		log.Printf("Failed to register stop sequence handler: %v", err)
		return
	}

	log.Println("NATS handlers registered successfully")

	// 注意：这里暂时不处理订阅清理，因为我们需要一个合适的context
	// TODO: 实现适当的订阅清理机制
}

// handleNATSPlayerRegister 处理NATS玩家注册
func (a *GameManagerActor) handleNATSPlayerRegister(msg *nats.Msg) {
	var playerReg common.MsgRegisterPlayer
	if err := json.Unmarshal(msg.Data, &playerReg); err != nil {
		log.Printf("Failed to unmarshal player register message: %v", err)
		return
	}

	// 通过Actor系统发送消息
	if playerPID, exists := a.players[playerReg.PlayerID]; exists {
		system := actor.NewActorSystem()
		system.Root.Send(playerPID, &playerReg)
	}
}

// handleNATSPlayerUnregister 处理NATS玩家注销
func (a *GameManagerActor) handleNATSPlayerUnregister(msg *nats.Msg) {
	var playerUnreg common.MsgUnregisterPlayer
	if err := json.Unmarshal(msg.Data, &playerUnreg); err != nil {
		log.Printf("Failed to unmarshal player unregister message: %v", err)
		return
	}

	// 通过Actor系统发送消息
	if playerPID, exists := a.players[playerUnreg.PlayerID]; exists {
		system := actor.NewActorSystem()
		system.Root.Send(playerPID, &playerUnreg)
	}
}

// handleNATSStartSequence 处理NATS开始序列
func (a *GameManagerActor) handleNATSStartSequence(msg *nats.Msg) {
	var startSeq common.MsgStartSequence
	if err := json.Unmarshal(msg.Data, &startSeq); err != nil {
		log.Printf("Failed to unmarshal start sequence message: %v", err)
		return
	}

	// 查找玩家Actor
	if playerPID, exists := a.players[startSeq.PlayerID]; exists {
		system := actor.NewActorSystem()
		system.Root.Send(playerPID, &startSeq)
	} else {
		log.Printf("Player not found for start sequence: %s", startSeq.PlayerID)
	}
}

// handleNATSStopSequence 处理NATS停止序列
func (a *GameManagerActor) handleNATSStopSequence(msg *nats.Msg) {
	var stopSeq common.MsgStopSequence
	if err := json.Unmarshal(msg.Data, &stopSeq); err != nil {
		log.Printf("Failed to unmarshal stop sequence message: %v", err)
		return
	}

	// 查找玩家Actor
	if playerPID, exists := a.players[stopSeq.PlayerID]; exists {
		system := actor.NewActorSystem()
		system.Root.Send(playerPID, &stopSeq)
	} else {
		log.Printf("Player not found for stop sequence: %s", stopSeq.PlayerID)
	}
}
