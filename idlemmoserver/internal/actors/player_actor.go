package actors

import (
	"time"

	gamedomain "idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

// PlayerActor 承担玩家会话的协调工作，将消息转交给命令与领域模型处理。
type PlayerActor struct {
	playerID     string
	root         *actor.RootContext
	conn         *websocket.Conn
	currentSeq   *actor.PID
	schedulerPID *actor.PID
	persistPID   *actor.PID
	model        *PlayerModel
	domain       *PlayerDomain
	commands     *CommandHandler
}

// NewPlayerActor 创建一个玩家 Actor，并初始化领域模型与命令分发器。
func NewPlayerActor(playerID string, root *actor.RootContext, persistPID *actor.PID, schedulerPID *actor.PID) actor.Actor {
	return &PlayerActor{
		playerID:     playerID,
		root:         root,
		schedulerPID: schedulerPID,
		persistPID:   persistPID,
		model:        NewPlayerModel(playerID),
		domain:       NewPlayerDomain(),
		commands:     NewCommandHandler(),
	}
}

// MsgAttachConn 用于 Gateway 告知玩家 Actor 绑定新的 WebSocket 连接。
type MsgAttachConn struct {
	Conn         *websocket.Conn
	RequestState bool
}

// MsgDetachConn 表示连接主动解除绑定。
type MsgDetachConn struct{}

// SeqResult 是 SequenceActor 向 PlayerActor 汇报的修炼结算消息。
type SeqResult struct {
	Gains        int64
	Rare         []string
	Items        []gamedomain.Item
	SeqID        string
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

// Receive 实现 protoactor-go 的消息收发入口。
func (p *PlayerActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *actor.Started:
		ctx.Send(p.persistPID, &MsgRegisterPlayer{PlayerID: p.playerID, PID: ctx.Self()})
		ctx.Send(p.persistPID, &MsgLoadPlayer{PlayerID: p.playerID, ReplyTo: ctx.Self()})
		p.model.LastActive = time.Now()

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
		p.model.MarkOffline(time.Now())
		logx.Info("player offline", "player", p.playerID, "limit", p.model.OfflineLimit)

	case *MsgPlayerReconnect:
		p.conn = m.Conn
		p.model.MarkOnline(time.Now())
		p.sendToClient(map[string]any{"type": "S_ReconnectOK"})

	case *MsgCheckExpire:
		if p.domain.ShouldExpire(p.model, time.Now()) {
			p.saveSnapshot(ctx)
			ctx.Send(p.persistPID, &MsgUnregisterPlayer{PlayerID: p.playerID})
			logx.Warn("player session expired", "player", p.playerID)
			ctx.Stop(ctx.Self())
		}

	case *MsgConnClosed:
		if m.Conn == p.conn {
			p.conn = nil
			p.model.MarkOffline(time.Now())
		}

	case *actor.Terminated:
		if p.currentSeq != nil && m.Who.Equal(p.currentSeq) {
			p.currentSeq = nil
			p.domain.OnSequenceStopped(p.model)
			if p.model.IsOnline {
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

// handleAttachConn 负责绑定新连接、结算离线收益并推送当前状态。
func (p *PlayerActor) handleAttachConn(msg *MsgAttachConn) {
	p.conn = msg.Conn
	now := time.Now()
	p.model.MarkOnline(now)

	logx.Info("收到 MsgAttachConn", "playerID", p.playerID, "requestState", msg.RequestState)

	if msg.RequestState {
		offlineGain, offlineItems, duration := p.domain.ApplyOfflineRewards(p.model, now)
		if offlineGain > 0 || len(offlineItems) > 0 {
			p.sendToClient(map[string]any{
				"type":             "S_OfflineReward",
				"gains":            offlineGain,
				"offline_duration": int64(duration.Seconds()),
				"offline_items":    offlineItems,
				"bag":              p.model.Inventory.List(),
			})
		}

		payload := p.domain.BuildReconnectedPayload(p.model, p.currentSeq != nil)
		logx.Info("发送 S_Reconnected 消息", "playerID", p.playerID, "payload", payload)
		p.sendToClient(payload)
	} else {
		p.sendToClient(map[string]any{"type": "S_Reconnected", "msg": "重连成功"})
	}
}

// handleLoadResult 将持久化层返回的数据转换为领域模型状态。
func (p *PlayerActor) handleLoadResult(m *MsgLoadResult) {
	if m.Err != nil || m.Data == nil {
		p.domain.InitNewPlayer(p.model)
		p.sendToClient(map[string]any{"type": "S_NewPlayer"})
		return
	}

	p.domain.ApplyLoadedData(p.model, m.Data)
	p.sendToClient(map[string]any{
		"type":                "S_LoadOK",
		"exp":                 p.model.Exp,
		"bag":                 p.model.Inventory.List(),
		"offline_limit_hours": p.model.OfflineLimitHours(),
		"equipment":           p.model.Equipment.Export(),
		"equipment_bonus":     p.model.Equipment.TotalBonus(),
	})
}

// handleClientPayload 将客户端数据交给命令处理器执行。
func (p *PlayerActor) handleClientPayload(ctx actor.Context, m *MsgClientPayload) {
	p.conn = m.Conn
	p.model.MarkOnline(time.Now())

	if err := p.commands.Handle(ctx, p, m); err != nil {
		logx.Warn("command execute failed", "player", p.playerID, "err", err)
		p.sendError(err.Error())
	}
}

// handleSeqResult 处理修炼结算结果并进行持久化。
func (p *PlayerActor) handleSeqResult(ctx actor.Context, m *SeqResult) {
	logx.Info("Player received SeqResult", "playerID", p.playerID, "seqID", m.SeqID, "gains", m.Gains, "items", len(m.Items), "isOnline", p.model.IsOnline)

	bagSnapshot := p.domain.ApplySequenceResult(p.model, m)
	if p.model.IsOnline && p.conn != nil {
		p.sendToClient(map[string]any{
			"type":            "S_SeqResult",
			"gains":           m.Gains,
			"rare":            m.Rare,
			"bag":             bagSnapshot,
			"seq_id":          m.SeqID,
			"level":           m.Level,
			"cur_exp":         m.CurExp,
			"leveled":         m.Leveled,
			"items":           m.Items,
			"sub_project_id":  m.SubProjectID,
			"equipment_bonus": p.model.Equipment.TotalBonus(),
		})
	}

	p.saveSnapshot(ctx)
}

// startSequence 校验并启动新的修炼序列。
func (p *PlayerActor) startSequence(ctx actor.Context, seqID, subProjectID string) error {
	if p.currentSeq != nil {
		p.stopCurrentSequence(ctx, false)
	}

	cfg, subProject, level, err := p.domain.PrepareSequenceStart(p.model, seqID, subProjectID)
	if err != nil {
		return err
	}

	bonus := p.model.Equipment.TotalBonus()
	pid := ctx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return NewSequenceActor(p.playerID, seqID, level, ctx.Self(), p.schedulerPID, subProject, bonus)
	}))
	ctx.Watch(pid)
	p.currentSeq = pid
	p.domain.OnSequenceStarted(p.model, seqID, subProject)

	interval := cfg.EffectiveInterval(subProject)
	p.sendToClient(map[string]any{
		"type":            "S_SeqStarted",
		"seq_id":          seqID,
		"level":           level,
		"sub_project_id":  p.model.ActiveSubProject,
		"tick_interval":   interval.Seconds(),
		"equipment_bonus": bonus,
	})
	return nil
}

// stopCurrentSequence 停止当前修炼，并视情况通知客户端。
func (p *PlayerActor) stopCurrentSequence(ctx actor.Context, notify bool) {
	if p.currentSeq == nil {
		return
	}
	ctx.Stop(p.currentSeq)
	p.currentSeq = nil
	p.domain.OnSequenceStopped(p.model)
	if notify && p.model.IsOnline {
		p.sendToClient(map[string]any{
			"type":               "S_SeqEnded",
			"is_running":         false,
			"seq_id":             "",
			"seq_level":          0,
			"active_sub_project": "",
		})
	}
}

// saveSnapshot 向持久化层发送保存请求。
func (p *PlayerActor) saveSnapshot(ctx actor.Context) {
	ctx.Send(p.persistPID, &MsgSavePlayer{
		PlayerID:          p.playerID,
		SeqLevels:         p.model.SeqLevels,
		Inventory:         p.model.Inventory,
		Exp:               p.model.Exp,
		Equipment:         p.model.Equipment.ExportState(),
		OfflineLimitHours: p.model.OfflineLimitHours(),
	})
}

// pushEquipmentBonus 在装备变化或新序列开启时同步装备加成给 SequenceActor。
func (p *PlayerActor) pushEquipmentBonus(ctx actor.Context) {
	if p.currentSeq != nil {
		ctx.Send(p.currentSeq, &MsgUpdateEquipmentBonus{Bonus: p.model.Equipment.TotalBonus()})
	}
}

// sendEquipmentState 向客户端发送当前装备信息，可选包含装备目录。
func (p *PlayerActor) sendEquipmentState(includeCatalog bool) {
	payload := map[string]any{
		"type":      "S_EquipmentState",
		"equipment": p.model.Equipment.Export(),
		"bonus":     p.model.Equipment.TotalBonus(),
		"bag":       p.model.Inventory.List(),
	}
	if includeCatalog {
		payload["catalog"] = gamedomain.GetEquipmentCatalogSummary()
	}
	p.sendToClient(payload)
}

// sendEquipmentChanged 将装备变化通知客户端。
func (p *PlayerActor) sendEquipmentChanged() {
	p.sendToClient(map[string]any{
		"type":      "S_EquipmentChanged",
		"equipment": p.model.Equipment.Export(),
		"bonus":     p.model.Equipment.TotalBonus(),
		"bag":       p.model.Inventory.List(),
	})
}

// send 将任意结构体写入 WebSocket。
func (p *PlayerActor) send(v any) {
	if p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}

// sendToClient 在玩家在线时推送消息，否则忽略。
func (p *PlayerActor) sendToClient(v any) {
	if p.model.IsOnline && p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}

// sendError 用统一格式返回错误消息。
func (p *PlayerActor) sendError(msg string) {
	p.sendToClient(map[string]any{"type": "S_Error", "msg": msg})
}
