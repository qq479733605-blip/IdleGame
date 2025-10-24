package actors

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type GatewayActor struct {
	root       *actor.RootContext
	persistPID *actor.PID

	connsMu sync.Mutex
	conns   map[*websocket.Conn]*actor.PID // 当前连接 → PlayerActor
	players sync.Map                       // playerID → *actor.PID（长存）
}

// ✅ 构造函数
func NewGatewayActor(root *actor.RootContext, persistPID *actor.PID) *GatewayActor {
	return &GatewayActor{
		root:       root,
		persistPID: persistPID,
		conns:      make(map[*websocket.Conn]*actor.PID),
	}
}

// ---------------------------
// 消息定义
// ---------------------------
type MsgNewConn struct {
	Conn  *websocket.Conn
	Token string
}

// ---------------------------
// 🚪 处理逻辑
// ---------------------------

func (g *GatewayActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {

	// 🟢 新连接（登录 + WebSocket 连接）
	case *MsgNewConn:
		playerID := parseToken(m.Token)

		// 优先使用缓存的 PlayerActor
		var playerPID *actor.PID
		if val, ok := g.players.Load(playerID); ok {
			playerPID = val.(*actor.PID)
			log.Printf("♻️ Reusing PlayerActor for %s", playerID)
		} else {
			// 不存在则创建新的 PlayerActor
			playerPID = g.ensurePlayerActor(ctx, playerID)
			g.players.Store(playerID, playerPID)
			log.Printf("🆕 Created PlayerActor for %s -> %v", playerID, playerPID)
		}

		// 绑定 Conn <-> Player
		g.connsMu.Lock()
		g.conns[m.Conn] = playerPID
		g.connsMu.Unlock()

		// 通知 PlayerActor 绑定新连接
		ctx.Send(playerPID, &MsgAttachConn{Conn: m.Conn})
		log.Printf("✅ Player %s attached WebSocket connection", playerID)

	// 💬 前端消息转发
	case *MsgFromWS:
		var base map[string]any
		if err := json.Unmarshal([]byte(m.Data), &base); err != nil {
			log.Printf("❌ Bad JSON: %v", err)
			return
		}

		msgType, _ := base["type"].(string)

		g.connsMu.Lock()
		pid := g.conns[m.Conn]
		g.connsMu.Unlock()

		if pid == nil {
			log.Printf("⚠️ No PlayerActor for message %s (conn lost mapping)", msgType)
			return
		}

		// 所有 C_ 开头的指令都自动转发
		if len(msgType) > 2 && msgType[:2] == "C_" {
			ctx.Send(pid, &MsgClientPayload{Conn: m.Conn, Raw: m.Data})
		} else {
			log.Printf("⚠️ Unknown msg type: %s", msgType)
		}

	// 🔌 连接断开（PlayerActor 保留）
	case *MsgConnClosed:
		g.connsMu.Lock()
		pid := g.conns[m.Conn]
		delete(g.conns, m.Conn)
		g.connsMu.Unlock()

		if pid != nil {
			ctx.Send(pid, &MsgDetachConn{}) // 通知 PlayerActor 清空连接
			log.Printf("🔌 Conn closed, PlayerActor %v kept alive", pid)
		}
	}
}

// ---------------------------
// 🧠 工具函数
// ---------------------------

// 创建 PlayerActor
func (g *GatewayActor) ensurePlayerActor(ctx actor.Context, playerID string) *actor.PID {
	props := actor.PropsFromProducer(func() actor.Actor {
		return NewPlayerActor(playerID, g.root, g.persistPID)
	})
	pid := g.root.Spawn(props)
	return pid
}

// 简单 token 解析
func parseToken(token string) string {
	if len(token) > 9 && token[:9] == "mock-jwt-" {
		return token[9:]
	}
	return token
}
