package actors

import (
	"encoding/json"
	"math/rand"
	"time"

	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type PlayerActor struct {
	playerID         string
	root             *actor.RootContext
	conn             *websocket.Conn
	currentSeq       *actor.PID
	currentSeqID     string
	seqLevels        map[string]int
	inventory        *domain.Inventory
	equipment        *domain.EquipmentLoadout
	exp              int64
	schedulerPID     *actor.PID
	activeSubProject string
	isOnline         bool
	offlineStart     time.Time
	offlineLimit     time.Duration
	lastActive       time.Time
	persistPID       *actor.PID
}

func NewPlayerActor(playerID string, root *actor.RootContext, persistPID *actor.PID, schedulerPID *actor.PID) actor.Actor {
	return &PlayerActor{
		playerID:     playerID,
		root:         root,
		seqLevels:    map[string]int{},
		inventory:    domain.NewInventory(200),
		equipment:    domain.NewEquipmentLoadout(),
		isOnline:     true,
		offlineLimit: 10 * time.Hour,
		persistPID:   persistPID,
		schedulerPID: schedulerPID,
	}
}

type reqStart struct {
	Type         string `json:"type"`
	SeqID        string `json:"seq_id"`
	Target       int64  `json:"target"`
	SubProjectID string `json:"sub_project_id"`
}

type MsgAttachConn struct {
	Conn         *websocket.Conn
	RequestState bool
}

type MsgDetachConn struct{}

type SeqResult struct {
	Gains        int64
	Rare         []string
	Items        []domain.Item
	SeqID        string
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

func (p *PlayerActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *actor.Started:
		ctx.Send(p.persistPID, &MsgRegisterPlayer{PlayerID: p.playerID, PID: ctx.Self()})
		ctx.Send(p.persistPID, &MsgLoadPlayer{PlayerID: p.playerID, ReplyTo: ctx.Self()})
		p.lastActive = time.Now()

	case *MsgAttachConn:
		p.handleAttachConn(m)

	case *MsgDetachConn:
		if p.conn != nil {
			p.conn = nil
			logx.Info("🕓 Player %s disconnected (actor retained)", p.playerID)
		}

	case *MsgLoadResult:
		p.handleLoadResult(m)

	case *MsgClientPayload:
		p.handleClientPayload(ctx, m)

	case *SeqResult:
		p.handleSeqResult(ctx, m)

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
		if !p.isOnline && !p.offlineStart.IsZero() && time.Since(p.offlineStart) > p.offlineLimit {
			ctx.Send(p.persistPID, &MsgSavePlayer{
				PlayerID:          p.playerID,
				SeqLevels:         p.seqLevels,
				Inventory:         p.inventory,
				Exp:               p.exp,
				OfflineLimitHours: int64(p.offlineLimit / time.Hour),
			})
			ctx.Send(p.persistPID, &MsgUnregisterPlayer{PlayerID: p.playerID})
			logx.Warn("player session expired", "player", p.playerID)
			ctx.Stop(ctx.Self())
		}

	case *MsgConnClosed:
		if m.Conn == p.conn {
			p.conn = nil
			p.isOnline = false
			p.offlineStart = time.Now()
		}

	case *actor.Terminated:
		if p.currentSeq != nil && m.Who.Equal(p.currentSeq) {
			p.currentSeq = nil
			p.currentSeqID = ""
			p.activeSubProject = ""
			if p.isOnline {
				p.sendToClient(map[string]any{
					"type":               "S_SeqEnded",
					"is_running":         false,
					"seq_id":             "",
					"seq_level":          0,
					"active_sub_project": "",
				})
			}
		}
	}
}

func (p *PlayerActor) handleAttachConn(msg *MsgAttachConn) {
	p.conn = msg.Conn
	p.isOnline = true
	p.lastActive = time.Now()

	logx.Info("收到 MsgAttachConn", "playerID", p.playerID, "requestState", msg.RequestState)

	if msg.RequestState {
		offlineGain, offlineItems, duration := p.calculateOfflineRewards()
		if offlineGain > 0 || len(offlineItems) > 0 {
			p.exp += offlineGain
			for itemID, count := range offlineItems {
				if err := p.inventory.AddItem(domain.Item{ID: itemID, Name: itemID}, count); err != nil {
					logx.Warn("offline reward add item failed", "playerID", p.playerID, "itemID", itemID, "count", count, "err", err)
				}
			}
			p.sendToClient(map[string]any{
				"type":             "S_OfflineReward",
				"gains":            offlineGain,
				"offline_duration": int64(duration.Seconds()),
				"offline_items":    offlineItems,
				"bag":              p.inventory.List(),
			})
		}

		payload := p.buildReconnectedPayload()
		logx.Info("发送 S_Reconnected 消息", "playerID", p.playerID, "payload", payload)
		p.sendToClient(payload)
	} else {
		p.sendToClient(map[string]any{"type": "S_Reconnected", "msg": "重连成功"})
	}

	p.offlineStart = time.Time{}
}

func (p *PlayerActor) calculateOfflineRewards() (int64, map[string]int64, time.Duration) {
	if p.offlineStart.IsZero() {
		return 0, map[string]int64{}, 0
	}
	duration := time.Since(p.offlineStart)
	if duration <= 0 || duration >= p.offlineLimit {
		return 0, map[string]int64{}, duration
	}

	gains := int64(0)
	items := make(map[string]int64)
	seconds := duration.Seconds()

	for seqID, level := range p.seqLevels {
		if level <= 0 {
			continue
		}
		cfg, exists := domain.GetSequenceConfig(seqID)
		if !exists || cfg == nil {
			continue
		}
		interval := cfg.TickInterval
		if interval <= 0 {
			interval = 1
		}
		ticks := int64(seconds / float64(interval))
		if ticks <= 0 {
			continue
		}

		gain := cfg.BaseGain + int64(float64(level)*cfg.GrowthFactor)
		gains += gain * ticks

		for _, drop := range cfg.Drops {
			if drop.DropChance <= 0 {
				continue
			}
			expected := float64(ticks) * drop.DropChance
			guaranteed := int64(expected)
			remainder := expected - float64(guaranteed)
			count := guaranteed
			if rand.Float64() < remainder {
				count++
			}
			if count > 0 {
				items[drop.ID] += count
			}
		}
	}

	return gains, items, duration
}

func (p *PlayerActor) buildReconnectedPayload() map[string]any {
	return map[string]any{
		"type":               "S_Reconnected",
		"msg":                "重连成功",
		"seq_id":             p.getCurrentSeqID(),
		"seq_level":          p.getCurrentSeqLevel(),
		"exp":                p.exp,
		"bag":                p.inventory.List(),
		"is_running":         p.currentSeq != nil,
		"seq_levels":         p.seqLevels,
		"equipment":          p.equipment.Export(),
		"equipment_bonus":    p.equipment.TotalBonus(),
		"active_sub_project": p.activeSubProject,
		// 装备配置现在在前端本地处理，不需要传输
	}
}

func (p *PlayerActor) handleLoadResult(m *MsgLoadResult) {
	if m.Err != nil || m.Data == nil {
		// 新玩家：为所有序列设置默认等级 1
		p.seqLevels = make(map[string]int)
		for seqID := range domain.Sequences {
			p.seqLevels[seqID] = 1 // 默认等级为 1
		}

		p.sendToClient(map[string]any{"type": "S_NewPlayer"})
		return
	}

	p.seqLevels = m.Data.SeqLevels
	for id, cnt := range m.Data.Inventory {
		_ = p.inventory.AddItem(domain.Item{ID: id, Name: id}, cnt)
	}
	p.exp = m.Data.Exp
	if m.Data.Equipment != nil {
		p.equipment.ImportState(m.Data.Equipment)
	}
	if m.Data.OfflineLimitHours > 0 {
		p.offlineLimit = time.Duration(m.Data.OfflineLimitHours) * time.Hour
	}

	p.sendToClient(map[string]any{
		"type":                "S_LoadOK",
		"exp":                 p.exp,
		"bag":                 p.inventory.List(),
		"offline_limit_hours": m.Data.OfflineLimitHours,
		"equipment":           p.equipment.Export(),
		"equipment_bonus":     p.equipment.TotalBonus(),
	})
}

func (p *PlayerActor) handleClientPayload(ctx actor.Context, m *MsgClientPayload) {
	p.conn = m.Conn
	p.isOnline = true
	p.lastActive = time.Now()

	var b baseMsg
	_ = json.Unmarshal(m.Raw, &b)

	switch b.Type {
	case "C_Login":
		p.sendToClient(map[string]any{
			"type":            "S_LoginOK",
			"msg":             "登录成功",
			"playerId":        p.playerID,
			"exp":             p.exp,
			"seq_levels":      p.seqLevels,
			"bag":             p.inventory.List(),
			"equipment":       p.equipment.Export(),
			"equipment_bonus": p.equipment.TotalBonus(),
			"is_running":      p.currentSeq != nil,
			"seq_id":          p.currentSeqID,
			"seq_level": func() int {
				if p.currentSeqID != "" {
					return p.seqLevels[p.currentSeqID]
				}
				return 0
			}(),
			"active_sub_project": p.activeSubProject,
		})

	case "C_StartSeq":
		p.handleStartSequence(ctx, m.Raw)

	// 配置现在在前端本地处理，不需要传输

	case "C_StopSeq":
		if p.currentSeq != nil {
			ctx.Stop(p.currentSeq)
			p.currentSeq = nil
			p.currentSeqID = ""
			p.activeSubProject = ""
			p.sendToClient(map[string]any{
				"type":               "S_SeqEnded",
				"is_running":         false,
				"seq_id":             "",
				"seq_level":          0,
				"active_sub_project": "",
			})
		}

	case "C_ListBag":
		p.sendToClient(map[string]any{"type": "S_BagInfo", "bag": p.inventory.List()})

	// 装备配置现在在前端本地处理，但装备状态仍需要传输
	case "C_ListEquipment":
		p.sendEquipmentState(false) // 不包含装备目录，只包含状态

	case "C_EquipItem":
		p.handleEquipItem(ctx, m.Raw)

	case "C_UnequipItem":
		p.handleUnequipItem(ctx, m.Raw)

	case "C_UseItem":
		var req struct {
			ItemID string `json:"item_id"`
			Count  int64  `json:"count"`
		}
		_ = json.Unmarshal(m.Raw, &req)
		if req.Count <= 0 {
			p.sendToClient(map[string]any{"type": "S_Error", "msg": "invalid count"})
			return
		}
		if err := p.inventory.RemoveItem(req.ItemID, req.Count); err != nil {
			p.sendToClient(map[string]any{"type": "S_Error", "msg": err.Error()})
			return
		}
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
			ItemID string `json:"item_id"`
			Count  int64  `json:"count"`
		}
		_ = json.Unmarshal(m.Raw, &req)
		if err := p.inventory.RemoveItem(req.ItemID, req.Count); err != nil {
			p.sendToClient(map[string]any{"type": "S_Error", "msg": err.Error()})
		} else {
			p.sendToClient(map[string]any{"type": "S_ItemRemoved", "item_id": req.ItemID, "count": req.Count})
		}
	}
}

func (p *PlayerActor) handleStartSequence(ctx actor.Context, raw []byte) {
	var req reqStart
	_ = json.Unmarshal(raw, &req)

	// 如果当前有运行的序列，自动停止它（支持无缝切换）
	if p.currentSeq != nil {
		ctx.Stop(p.currentSeq)
		p.currentSeq = nil
		p.currentSeqID = ""
		p.activeSubProject = ""
		// 注意：这里不发送 S_SeqEnded，因为马上会发送 S_SeqStarted
	}

	cfg, exists := domain.GetSequenceConfig(req.SeqID)
	if !exists {
		p.sendToClient(map[string]any{"type": "S_Error", "msg": "sequence not found"})
		return
	}

	level := p.seqLevels[req.SeqID]
	var subProject *domain.SequenceSubProject
	if req.SubProjectID != "" {
		sp, ok := cfg.GetSubProject(req.SubProjectID)
		if !ok {
			p.sendToClient(map[string]any{"type": "S_Error", "msg": "sub project not found"})
			return
		}
		if level < sp.UnlockLevel {
			p.sendToClient(map[string]any{"type": "S_Error", "msg": "子项目未解锁"})
			return
		}
		subProject = sp
	}

	bonus := p.equipment.TotalBonus()
	pid := ctx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return NewSequenceActor(p.playerID, req.SeqID, level, ctx.Self(), p.schedulerPID, subProject, bonus)
	}))
	ctx.Watch(pid)
	p.currentSeq = pid
	p.currentSeqID = req.SeqID
	if subProject != nil {
		p.activeSubProject = subProject.ID
	} else {
		p.activeSubProject = ""
	}

	interval := cfg.EffectiveInterval(subProject)
	p.sendToClient(map[string]any{
		"type":            "S_SeqStarted",
		"seq_id":          req.SeqID,
		"level":           level,
		"sub_project_id":  p.activeSubProject,
		"tick_interval":   interval.Seconds(),
		"equipment_bonus": bonus,
	})
}

func (p *PlayerActor) handleSeqResult(ctx actor.Context, m *SeqResult) {
	logx.Info("Player received SeqResult", "playerID", p.playerID, "seqID", m.SeqID, "gains", m.Gains, "items", len(m.Items), "isOnline", p.isOnline)

	for _, it := range m.Items {
		if err := p.inventory.AddItem(it, 1); err != nil {
			logx.Error("Failed to add item", "itemID", it.ID, "error", err)
		}
	}

	if m.SeqID != "" {
		p.seqLevels[m.SeqID] = m.Level
	}
	p.activeSubProject = m.SubProjectID
	p.exp += m.Gains

	currentBag := p.inventory.List()
	if p.isOnline && p.conn != nil {
		p.sendToClient(map[string]any{
			"type":            "S_SeqResult",
			"gains":           m.Gains,
			"rare":            m.Rare,
			"bag":             currentBag,
			"seq_id":          m.SeqID,
			"level":           m.Level,
			"cur_exp":         m.CurExp,
			"leveled":         m.Leveled,
			"items":           m.Items,
			"sub_project_id":  m.SubProjectID,
			"equipment_bonus": p.equipment.TotalBonus(),
		})
	}

	ctx.Send(p.persistPID, &MsgSavePlayer{
		PlayerID:          p.playerID,
		SeqLevels:         p.seqLevels,
		Inventory:         p.inventory,
		Exp:               p.exp,
		Equipment:         p.equipment.ExportState(),
		OfflineLimitHours: int64(p.offlineLimit / time.Hour),
	})
}

func (p *PlayerActor) handleEquipItem(ctx actor.Context, raw []byte) {
	var req struct {
		ItemID      string `json:"item_id"`
		Enhancement int    `json:"enhancement"`
	}
	_ = json.Unmarshal(raw, &req)
	if req.ItemID == "" {
		p.sendToClient(map[string]any{"type": "S_Error", "msg": "请选择要装备的物品"})
		return
	}

	def, ok := domain.GetEquipmentDefinition(req.ItemID)
	if !ok {
		p.sendToClient(map[string]any{"type": "S_Error", "msg": "该物品无法装备"})
		return
	}

	if err := p.inventory.RemoveItem(req.ItemID, 1); err != nil {
		p.sendToClient(map[string]any{"type": "S_Error", "msg": err.Error()})
		return
	}

	replaced := p.equipment.Equip(def, req.Enhancement)
	if replaced != nil {
		if err := p.inventory.AddItem(domain.Item{ID: replaced.Definition.ID, Name: replaced.Definition.Name}, 1); err != nil {
			p.equipment.Equip(replaced.Definition, replaced.Enhancement)
			_ = p.inventory.AddItem(domain.Item{ID: def.ID, Name: def.Name}, 1)
			p.sendToClient(map[string]any{"type": "S_Error", "msg": "背包空间不足"})
			return
		}
	}

	logx.Info("equip item", "player", p.playerID, "item", def.ID)
	p.pushEquipmentBonus(ctx)
	p.sendEquipmentChanged()
}

func (p *PlayerActor) handleUnequipItem(ctx actor.Context, raw []byte) {
	var req struct {
		Slot string `json:"slot"`
	}
	_ = json.Unmarshal(raw, &req)
	if req.Slot == "" {
		p.sendToClient(map[string]any{"type": "S_Error", "msg": "请选择要卸下的位置"})
		return
	}

	slot := domain.EquipmentSlot(req.Slot)
	item := p.equipment.Unequip(slot)
	if item == nil {
		p.sendToClient(map[string]any{"type": "S_Error", "msg": "该位置没有装备"})
		return
	}

	if err := p.inventory.AddItem(domain.Item{ID: item.Definition.ID, Name: item.Definition.Name}, 1); err != nil {
		p.equipment.Equip(item.Definition, item.Enhancement)
		p.sendToClient(map[string]any{"type": "S_Error", "msg": "背包空间不足"})
		return
	}

	logx.Info("unequip item", "player", p.playerID, "slot", slot)
	p.pushEquipmentBonus(ctx)
	p.sendEquipmentChanged()
}

func (p *PlayerActor) sendEquipmentState(includeCatalog bool) {
	payload := map[string]any{
		"type":      "S_EquipmentState",
		"equipment": p.equipment.Export(),
		"bonus":     p.equipment.TotalBonus(),
		"bag":       p.inventory.List(),
	}
	if includeCatalog {
		payload["catalog"] = domain.GetEquipmentCatalogSummary()
	}
	p.sendToClient(payload)
}

func (p *PlayerActor) sendEquipmentChanged() {
	p.sendToClient(map[string]any{
		"type":      "S_EquipmentChanged",
		"equipment": p.equipment.Export(),
		"bonus":     p.equipment.TotalBonus(),
		"bag":       p.inventory.List(),
	})
}

func (p *PlayerActor) pushEquipmentBonus(ctx actor.Context) {
	if p.currentSeq != nil {
		ctx.Send(p.currentSeq, &MsgUpdateEquipmentBonus{Bonus: p.equipment.TotalBonus()})
	}
}

func (p *PlayerActor) send(v any) {
	if p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}

func (p *PlayerActor) sendToClient(v any) {
	if p.isOnline && p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}

func (p *PlayerActor) getCurrentSeqID() string {
	return p.currentSeqID
}

func (p *PlayerActor) getCurrentSeqLevel() int {
	if p.currentSeqID != "" {
		return p.seqLevels[p.currentSeqID]
	}
	return 0
}
