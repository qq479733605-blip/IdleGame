package actors

import (
	"encoding/json"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type GatewayActor struct {
	root    *actor.RootContext
	connsMu sync.Mutex
	conns   map[*websocket.Conn]*actor.PID // 每连接 -> PlayerActor
}

func NewGatewayActor(root *actor.RootContext) *GatewayActor {
	return &GatewayActor{root: root, conns: make(map[*websocket.Conn]*actor.PID)}
}

// 来自 WS 的原始消息
type MsgFromWS struct {
	Conn *websocket.Conn
	Data []byte
}

type MsgWSClosed struct{ Conn *websocket.Conn }

type baseMsg struct {
	Type string `json:"type"`
}

type CLogin struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type CStartSeq struct {
	Type   string `json:"type"`
	SeqID  string `json:"seq_id"`
	Target int64  `json:"target"`
}

type CStopSeq struct {
	Type string `json:"type"`
}

func (g *GatewayActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *MsgFromWS:
		var b baseMsg
		_ = json.Unmarshal(m.Data, &b)
		switch b.Type {
		case "C_Login":
			// TODO: 校验 m.Token
			playerID := "player-" // 根据 token 解析玩家 ID（示例）
			pid := g.ensurePlayerActor(ctx, playerID)
			g.bindConn(m.Conn, pid)
			_ = m.Conn.WriteJSON(map[string]any{"type": "S_LoginOK"})

		case "C_StartSeq":
			g.connsMu.Lock()
			pid := g.conns[m.Conn]
			g.connsMu.Unlock()
			if pid != nil {
				ctx.Send(pid, &MsgClientPayload{Conn: m.Conn, Raw: m.Data})
			}

		case "C_StopSeq":
			g.connsMu.Lock()
			pid := g.conns[m.Conn]
			g.connsMu.Unlock()
			if pid != nil {
				ctx.Send(pid, &MsgClientPayload{Conn: m.Conn, Raw: m.Data})
			}
		}

	case *MsgWSClosed:
		g.connsMu.Lock()
		pid := g.conns[m.Conn]
		delete(g.conns, m.Conn)
		g.connsMu.Unlock()
		if pid != nil {
			ctx.Send(pid, &MsgConnClosed{Conn: m.Conn})
		}
	}
}

func (g *GatewayActor) ensurePlayerActor(ctx actor.Context, playerID string) *actor.PID {
	props := actor.PropsFromProducer(func() actor.Actor { return NewPlayerActor(playerID, g.root) })
	return g.root.Spawn(props)
}

func (g *GatewayActor) bindConn(conn *websocket.Conn, pid *actor.PID) {
	g.connsMu.Lock()
	g.conns[conn] = pid
	g.connsMu.Unlock()
}
