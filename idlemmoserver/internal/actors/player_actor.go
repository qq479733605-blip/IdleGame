package actors

import (
	"encoding/json"
	"idlemmoserver/internal/domain"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type PlayerActor struct {
	playerID   string
	root       *actor.RootContext
	conn       *websocket.Conn
	currentSeq *actor.PID
	seqLevels  map[string]int // 每种序列的等级
	inventory  *domain.Inventory
}

func NewPlayerActor(playerID string, root *actor.RootContext) actor.Actor {
	return &PlayerActor{
		playerID:  playerID,
		root:      root,
		seqLevels: map[string]int{},
		inventory: domain.NewInventory(100), // 100种物品上限
	}
}

type MsgClientPayload struct {
	Conn *websocket.Conn
	Raw  []byte
}

type MsgConnClosed struct{ Conn *websocket.Conn }

type SeqResult struct {
	Gains int64
	Rare  []string
	Items []domain.Item
}

type SeqStop struct{}

type reqStart struct {
	Type   string `json:"type"`
	SeqID  string `json:"seq_id"`
	Target int64  `json:"target"`
}

type reqStop struct {
	Type string `json:"type"`
}

func (p *PlayerActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *MsgClientPayload:
		p.conn = m.Conn
		var b baseMsg
		_ = json.Unmarshal(m.Raw, &b)
		switch b.Type {
		case "C_StartSeq":
			var req reqStart
			_ = json.Unmarshal(m.Raw, &req)
			if p.currentSeq != nil {
				p.send(map[string]any{"type": "S_Err", "msg": "sequence running"})
				return
			}
			level := p.seqLevels[req.SeqID]
			pid := ctx.Spawn(actor.PropsFromProducer(func() actor.Actor {
				return NewSequenceActor(p.playerID, req.SeqID, level, ctx.Self())
			}))
			p.currentSeq = pid
			p.send(map[string]any{"type": "S_SeqStarted", "seq_id": req.SeqID})

		case "C_StopSeq":
			if p.currentSeq != nil {
				ctx.Send(p.currentSeq, &SeqStop{})
			}
		}

	case *SeqResult:
		// 增加背包物品
		for _, item := range m.Items {
			_ = p.inventory.AddItem(item, 1)
		}

		// 汇总当前背包列表
		bag := p.inventory.List()

		// 向客户端返回结算信息
		p.send(map[string]any{
			"type":  "S_SeqResult",
			"gains": m.Gains,
			"rare":  m.Rare,
			"bag":   bag,
		})

	case *MsgConnClosed:
		if m.Conn == p.conn {
			p.conn = nil
		}
	}
}

func (p *PlayerActor) send(v any) {
	if p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}
