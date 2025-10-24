package actors

import (
	"encoding/json"
	"idlemmoserver/internal/logx"
	"math/rand"
	"time"

	"idlemmoserver/internal/domain"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type PlayerActor struct {
	playerID     string
	root         *actor.RootContext
	conn         *websocket.Conn
	currentSeq   *actor.PID
	currentSeqID string // 添加当前序列ID跟踪
	seqLevels    map[string]int
	inventory    *domain.Inventory
	exp          int64

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
	Conn         *websocket.Conn
	RequestState bool // 是否请求当前状态
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
		p.isOnline = true
		p.lastActive = time.Now()

		logx.Info("收到 MsgAttachConn", "playerID", p.playerID, "requestState", m.RequestState)

		if m.RequestState {
			// 请求当前状态，发送完整状态
			offlineDuration := time.Since(p.offlineStart).Seconds()
			logx.Info("计算离线时长", "playerID", p.playerID, "offlineDuration", offlineDuration)

			// 如果离线时间太长，计算离线收益
			var offlineGains int64
			offlineItems := make(map[string]int64)

			if offlineDuration > 0 && offlineDuration < float64(p.offlineLimit.Seconds()) {
				// 根据序列等级计算离线收益
				for seqID, level := range p.seqLevels {
					cfg, exists := domain.GetSequenceConfig(seqID)
					if exists && level > 0 {
						// 简单的离线收益计算：基础收益 + 等级加成
						gain := cfg.BaseGain + int64(float64(level)*cfg.GrowthFactor)
						ticks := int64(offlineDuration) / int64(cfg.TickInterval)
						offlineGains += gain * ticks

						// 计算掉落物品（简化版）
						dropChance := float64(ticks) * 0.3 // 每10次tick掉落1个物品
						if dropChance >= 1 {
							for _, item := range cfg.Drops {
								if rand.Float64() < 0.5 { // 50%概率掉落每种物品
									offlineItems[item.ID] += 1
								}
							}
						}
					}
				}

				// 更新玩家状态
				p.exp += offlineGains
				for itemID, count := range offlineItems {
					for i := int64(0); i < count; i++ {
						p.inventory.AddItem(domain.Item{ID: itemID, Name: itemID}, 1)
					}
				}

				// 发送离线收益信息
				p.sendToClient(map[string]any{
					"type":             "S_OfflineReward",
					"gains":            offlineGains,
					"offline_duration": int64(offlineDuration),
					"offline_items":    offlineItems,
				})
			}

			// 发送当前状态
			reconnectedMsg := map[string]any{
				"type":       "S_Reconnected",
				"msg":        "重连成功",
				"seq_id":     p.getCurrentSeqID(),
				"seq_level":  p.getCurrentSeqLevel(),
				"exp":        p.exp,
				"bag":        p.inventory.List(),
				"is_running": p.currentSeq != nil,
				"seq_levels": p.seqLevels, // 发送所有序列的等级信息
			}
			logx.Info("发送 S_Reconnected 消息", "playerID", p.playerID, "msg", reconnectedMsg)
			p.sendToClient(reconnectedMsg)
		} else {
			p.sendToClient(map[string]any{
				"type": "S_Reconnected",
				"msg":  "重连成功",
			})
		}

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
			p.sendToClient(map[string]any{"type": "S_LoadOK", "exp": p.exp, "bag": p.inventory.List(), "offline_limit_hours": m.Data.OfflineLimitHours})
		} else {
			p.sendToClient(map[string]any{"type": "S_NewPlayer"})
		}

	case *MsgClientPayload:
		p.conn = m.Conn
		p.isOnline = true
		p.lastActive = time.Now()

		var b baseMsg
		_ = json.Unmarshal(m.Raw, &b)
		switch b.Type {
		case "C_Login":
			// 确认登录状态，返回玩家信息
			p.sendToClient(map[string]any{
				"type":     "S_LoginOK",
				"msg":      "登录成功",
				"playerId": p.playerID,
				"exp":      p.exp,
			})

		case "C_StartSeq":
			var req reqStart
			_ = json.Unmarshal(m.Raw, &req)
			if p.currentSeq != nil {
				p.sendToClient(map[string]any{"type": "S_Err", "msg": "sequence running"})
				return
			}
			level := p.seqLevels[req.SeqID]
			pid := ctx.Spawn(actor.PropsFromProducer(func() actor.Actor {
				return NewSequenceActor(p.playerID, req.SeqID, level, ctx.Self())
			}))
			p.currentSeq = pid
			p.currentSeqID = req.SeqID // 设置当前序列ID
			p.sendToClient(map[string]any{"type": "S_SeqStarted", "seq_id": req.SeqID, "level": level})

		case "C_ListSeq":
			seqs := domain.GetAllSequences()
			p.sendToClient(map[string]any{
				"type":      "S_ListSeq",
				"sequences": seqs,
			})

		case "C_StopSeq":
			if p.currentSeq != nil {
				ctx.Stop(p.currentSeq)
				p.currentSeq = nil
				p.currentSeqID = "" // 清空当前序列ID
				p.sendToClient(map[string]any{"type": "S_SeqEnded"})
			}
		case "C_ListBag":
			p.sendToClient(map[string]any{
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
				p.sendToClient(map[string]any{"type": "S_Error", "msg": "invalid count"})
				return
			}

			err := p.inventory.RemoveItem(req.ItemID, req.Count)
			if err != nil {
				p.sendToClient(map[string]any{"type": "S_Error", "msg": err.Error()})
				return
			}

			// 简单示例：使用物品增加经验
			p.exp += req.Count * 10
			p.sendToClient(map[string]any{
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
				p.sendToClient(map[string]any{"type": "S_Error", "msg": err.Error()})
			} else {
				p.sendToClient(map[string]any{"type": "S_ItemRemoved", "item_id": req.ItemID, "count": req.Count})
			}
		}

	case *SeqResult:
		logx.Info("Player received SeqResult", "playerID", p.playerID, "seqID", m.SeqID,
			"gains", m.Gains, "items", len(m.Items), "isOnline", p.isOnline)

		// 背包入库
		for _, it := range m.Items {
			logx.Info("Adding item to inventory", "itemID", it.ID, "itemName", it.Name)
			err := p.inventory.AddItem(it, 1)
			if err != nil {
				logx.Error("Failed to add item", "itemID", it.ID, "error", err)
			} else {
				logx.Info("Item added successfully", "itemID", it.ID)
			}
		}
		// 成长同步
		if m.SeqID != "" {
			p.seqLevels[m.SeqID] = m.Level
		}
		p.exp += m.Gains

		// 获取当前背包状态
		currentBag := p.inventory.List()
		logx.Info("Current inventory", "playerID", p.playerID, "bag", currentBag)

		// UI 只在在线时返回
		if p.isOnline && p.conn != nil {
			logx.Info("Sending S_SeqResult to client", "playerID", p.playerID)
			p.sendToClient(map[string]any{
				"type":    "S_SeqResult",
				"gains":   m.Gains,
				"rare":    m.Rare,
				"bag":     currentBag,
				"seq_id":  m.SeqID,
				"level":   m.Level,
				"cur_exp": m.CurExp,
				"leveled": m.Leveled,
			})
		} else {
			logx.Warn("Player not online or no connection", "playerID", p.playerID, "isOnline", p.isOnline, "hasConn", p.conn != nil)
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
		p.sendToClient(map[string]any{"type": "S_ReconnectOK"})

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

// 只在在线时发送消息给客户端
func (p *PlayerActor) sendToClient(v any) {
	if p.isOnline && p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}

// 辅助方法：获取当前运行的序列ID
func (p *PlayerActor) getCurrentSeqID() string {
	return p.currentSeqID
}

// 辅助方法：获取当前序列等级
func (p *PlayerActor) getCurrentSeqLevel() int {
	if p.currentSeqID != "" {
		return p.seqLevels[p.currentSeqID]
	}
	return 0
}
