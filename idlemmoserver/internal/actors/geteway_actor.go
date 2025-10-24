package actors

import (
	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

// GatewayActor 简化的网关Actor，只负责PlayerActor的生命周期管理
type GatewayActor struct {
	root       *actor.RootContext
	persistPID *actor.PID
	players    sync.Map // playerID -> *actor.PID（长存）
}

// NewGatewayActor 创建GatewayActor
func NewGatewayActor(root *actor.RootContext, persistPID *actor.PID) *GatewayActor {
	return &GatewayActor{
		root:       root,
		persistPID: persistPID,
	}
}

// MsgEnsurePlayer 确保PlayerActor存在
type MsgEnsurePlayer struct {
	PlayerID string
	ReplyTo  *actor.PID
}

// MsgPlayerReady PlayerActor准备就绪
type MsgPlayerReady struct {
	PlayerPID *actor.PID
}

// ---------------------------
// 消息处理逻辑
// ---------------------------

func (g *GatewayActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {

	case *actor.Started:
		log.Printf("🚀 GatewayActor started")

	case *MsgEnsurePlayer:
		// 确保PlayerActor存在并返回
		playerPID := g.ensurePlayerActor(ctx, m.PlayerID)

		if m.ReplyTo != nil {
			ctx.Respond(&MsgPlayerReady{PlayerPID: playerPID})
		}

		log.Printf("✅ PlayerActor ensured for %s -> %v", m.PlayerID, playerPID)

	case *MsgUnregisterPlayer:
		// 移除PlayerActor
		g.players.Delete(m.PlayerID)
		log.Printf("🗑️ PlayerActor unregistered for %s", m.PlayerID)

	default:
		log.Printf("⚠️ GatewayActor received unknown message type: %T", m)
	}
}

// ---------------------------
// 工具函数
// ---------------------------

// ensurePlayerActor 确保PlayerActor存在
func (g *GatewayActor) ensurePlayerActor(ctx actor.Context, playerID string) *actor.PID {
	// 检查是否已存在
	if val, ok := g.players.Load(playerID); ok {
		return val.(*actor.PID)
	}

	// 创建新的PlayerActor
	props := actor.PropsFromProducer(func() actor.Actor {
		return NewPlayerActor(playerID, g.root, g.persistPID)
	})
	pid := ctx.Spawn(props)
	g.players.Store(playerID, pid)

	log.Printf("🆕 Created new PlayerActor for %s -> %v", playerID, pid)
	return pid
}
