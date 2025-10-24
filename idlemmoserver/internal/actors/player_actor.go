package actors

import (
	"encoding/json"
	"idlemmoserver/internal/logx"
	"time"

	"idlemmoserver/internal/domain"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type PlayerActor struct {
	playerID   string
	root       *actor.RootContext
	conn       *websocket.Conn
	currentSeq *actor.PID
	seqLevels  map[string]int
	inventory  *domain.Inventory
	exp        int64

	// 离线机制
	isOnline     bool
	offlineStart time.Time
	offlineLimit time.Duration // 默认 10 小时，可持久化
	lastActive   time.Time

	// 持久化
	persistPID *actor.PID
}

func NewPlayerActor(playerID string, root *actor.RootContext, persistPID *actor.PID) actor.Actor {
	return &PlayerActor{
		playerID:     playerID,
		root:         root,
		seqLevels:    map[string]int{},
		inventory:    domain.NewInventory(200),
		isOnline:     true,
		offlineLimit: 10 * time.Hour, // 默认 10 小时
		persistPID:   persistPID,
	}
}

type reqStart struct {
	Type   string `json:"type"`
	SeqID  string `json:"seq_id"`
	Target int64  `json:"target"`
}
type reqStop struct {
	Type string `json:"type"`
}

// MsgAttachConn 玩家重新连接
type MsgAttachConn struct {
	Conn *websocket.Conn
}

// MsgDetachConn 玩家断线
type MsgDetachConn struct{}

type SeqResult struct {
	Gains int64
	Rare  []string
	Items []domain.Item

	// 成长同步
	SeqID   string
	Level   int
	CurExp  int64
	Leveled bool
}

func (p *PlayerActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {

	case *actor.Started:
		// 1) 注册到 PersistActor，并加载存档
		ctx.Send(p.persistPID, &MsgRegisterPlayer{PlayerID: p.playerID, PID: ctx.Self()})
		ctx.Send(p.persistPID, &MsgLoadPlayer{PlayerID: p.playerID, ReplyTo: ctx.Self()})
		p.lastActive = time.Now()
	case *MsgAttachConn:
		p.conn = m.Conn
		p.send(map[string]any{
			"type": "S_Reconnected",
			"msg":  "重连成功",
		})

	case *MsgDetachConn:
		if p.conn != nil {
			p.conn = nil
			logx.Info("🕓 Player %s disconnected (actor retained)", p.playerID)
		}
	case *MsgLoadResult:
		if m.Err == nil && m.Data != nil {
			p.seqLevels = m.Data.SeqLevels
			for id, cnt := range m.Data.Inventory {
				_ = p.inventory.AddItem(domain.Item{ID: id, Name: id}, cnt)
			}
			p.exp = m.Data.Exp
			if m.Data.OfflineLimitHours > 0 {
				p.offlineLimit = time.Duration(m.Data.OfflineLimitHours) * time.Hour
			}
			p.send(map[string]any{"type": "S_LoadOK", "exp": p.exp, "bag": p.inventory.List(), "offline_limit_hours": m.Data.OfflineLimitHours})
		} else {
			p.send(map[string]any{"type": "S_NewPlayer"})
		}

	case *MsgClientPayload:
		p.conn = m.Conn
		p.isOnline = true
		p.lastActive = time.Now()

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
			p.send(map[string]any{"type": "S_SeqStarted", "seq_id": req.SeqID, "level": level})

		case "C_ListSeq":
			seqs := domain.GetAllSequences()
			p.send(map[string]any{
				"type":      "S_ListSeq",
				"sequences": seqs,
			})

		case "C_StopSeq":
			if p.currentSeq != nil {
				ctx.Stop(p.currentSeq)
				p.currentSeq = nil
				p.send(map[string]any{"type": "S_SeqEnded"})
			}
		case "C_ListBag":
			p.send(map[string]any{
				"type": "S_BagInfo",
				"bag":  p.inventory.List(),
			})

		case "C_UseItem":
			var req struct {
				Type   string `json:"type"`
				ItemID string `json:"item_id"`
				Count  int64  `json:"count"`
			}
			_ = json.Unmarshal(m.Raw, &req)
			if req.Count <= 0 {
				p.send(map[string]any{"type": "S_Error", "msg": "invalid count"})
				return
			}

			err := p.inventory.RemoveItem(req.ItemID, req.Count)
			if err != nil {
				p.send(map[string]any{"type": "S_Error", "msg": err.Error()})
				return
			}

			// 简单示例：使用物品增加经验
			p.exp += req.Count * 10
			p.send(map[string]any{
				"type":    "S_ItemUsed",
				"item_id": req.ItemID,
				"count":   req.Count,
				"effect":  "exp+10",
				"exp":     p.exp,
			})

		case "C_RemoveItem":
			var req struct {
				Type   string `json:"type"`
				ItemID string `json:"item_id"`
				Count  int64  `json:"count"`
			}
			_ = json.Unmarshal(m.Raw, &req)
			err := p.inventory.RemoveItem(req.ItemID, req.Count)
			if err != nil {
				p.send(map[string]any{"type": "S_Error", "msg": err.Error()})
			} else {
				p.send(map[string]any{"type": "S_ItemRemoved", "item_id": req.ItemID, "count": req.Count})
			}
		}

	case *SeqResult:
		// 背包入库
		for _, it := range m.Items {
			_ = p.inventory.AddItem(it, 1)
		}
		// 成长同步
		if m.SeqID != "" {
			p.seqLevels[m.SeqID] = m.Level
		}
		p.exp += m.Gains

		// UI 只在在线时返回
		if p.isOnline && p.conn != nil {
			p.send(map[string]any{
				"type":    "S_SeqResult",
				"gains":   m.Gains,
				"rare":    m.Rare,
				"bag":     p.inventory.List(),
				"seq_id":  m.SeqID,
				"level":   m.Level,
				"cur_exp": m.CurExp,
				"leveled": m.Leveled,
			})
		}

		// 异步存盘
		ctx.Send(p.persistPID, &MsgSavePlayer{
			PlayerID:          p.playerID,
			SeqLevels:         p.seqLevels,
			Inventory:         p.inventory,
			Exp:               p.exp,
			OfflineLimitHours: int64(p.offlineLimit / time.Hour),
		})

	case *MsgPlayerOffline:
		p.isOnline = false
		p.offlineStart = time.Now()
		logx.Info("player offline", "player", p.playerID, "limit", p.offlineLimit)

	case *MsgPlayerReconnect:
		p.isOnline = true
		p.conn = m.Conn
		p.lastActive = time.Now()
		p.send(map[string]any{"type": "S_ReconnectOK"})

	case *MsgCheckExpire:
		if !p.isOnline && time.Since(p.offlineStart) > p.offlineLimit {
			// 超时：存盘 & 结束
			ctx.Send(p.persistPID, &MsgSavePlayer{
				PlayerID:          p.playerID,
				SeqLevels:         p.seqLevels,
				Inventory:         p.inventory,
				Exp:               p.exp,
				OfflineLimitHours: int64(p.offlineLimit / time.Hour),
			})
			ctx.Send(p.persistPID, &MsgUnregisterPlayer{PlayerID: p.playerID})
			logx.Warn("inventory full", "player", p.playerID)
			ctx.Stop(ctx.Self())
		}

	case *MsgConnClosed:
		if m.Conn == p.conn {
			p.conn = nil
			p.isOnline = false
			p.offlineStart = time.Now()
		}
	}
}

func (p *PlayerActor) send(v any) {
	if p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}
