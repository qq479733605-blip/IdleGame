package actors

import (
	"encoding/json"
	"time"

	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type PlayerActor struct {
	playerID     string
	root         *actor.RootContext
	conn         *websocket.Conn
	currentSeq   *actor.PID
	schedulerPID *actor.PID
	persistPID   *actor.PID

	domain   *domain.PlayerDomain
	commands *domain.CommandHandler
}

func NewPlayerActor(playerID string, root *actor.RootContext, persistPID *actor.PID, schedulerPID *actor.PID) actor.Actor {
	service := domain.NewPlayerDomain(playerID)
	return &PlayerActor{
		playerID:     playerID,
		root:         root,
		schedulerPID: schedulerPID,
		persistPID:   persistPID,
		domain:       service,
		commands:     domain.NewPlayerCommandHandler(service),
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
		p.domain.MarkOffline(time.Now())
		logx.Info("player offline", "player", p.playerID)

	case *MsgPlayerReconnect:
		p.conn = m.Conn
		payload := p.domain.OnReconnect()
		p.sendToClient(payload)

	case *MsgCheckExpire:
		if p.domain.ShouldExpire(time.Now()) {
			p.saveState(ctx)
			ctx.Send(p.persistPID, &MsgUnregisterPlayer{PlayerID: p.playerID})
			logx.Warn("player session expired", "player", p.playerID)
			ctx.Stop(ctx.Self())
		}

	case *MsgConnClosed:
		if m.Conn == p.conn {
			p.conn = nil
			p.domain.MarkOffline(time.Now())
		}

	case *actor.Terminated:
		if p.currentSeq != nil && m.Who.Equal(p.currentSeq) {
			p.currentSeq = nil
			p.domain.ClearSequenceState()
			if p.domain.Model().IsOnline {
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
	logx.Info("收到 MsgAttachConn", "playerID", p.playerID, "requestState", msg.RequestState)
	result := p.domain.AttachConnection(msg.RequestState)

	for _, err := range result.InventoryErrors {
		logx.Warn("offline reward add item failed", "playerID", p.playerID, "error", err)
	}
	if reward := result.OfflineReward; reward != nil && (reward.Gains > 0 || len(reward.Items) > 0) {
		p.sendToClient(map[string]any{
			"type":             "S_OfflineReward",
			"gains":            reward.Gains,
			"offline_duration": int64(reward.Duration.Seconds()),
			"offline_items":    reward.Items,
			"bag":              p.domain.Model().Inventory.List(),
		})
	}
	if result.ReconnectedPayload != nil {
		p.sendToClient(result.ReconnectedPayload)
	}
}

func (p *PlayerActor) handleLoadResult(m *MsgLoadResult) {
	if m.Err != nil {
		logx.Error("load player failed", "player", p.playerID, "err", m.Err)
	}
	var snapshot *domain.PlayerSnapshot
	if m.Err == nil && m.Data != nil {
		snapshot = &domain.PlayerSnapshot{
			SeqLevels:         m.Data.SeqLevels,
			Inventory:         m.Data.Inventory,
			Exp:               m.Data.Exp,
			Equipment:         m.Data.Equipment,
			OfflineLimitHours: m.Data.OfflineLimitHours,
		}
	}
	outcome := p.domain.ApplySnapshot(snapshot)
	if outcome != nil {
		for _, payload := range outcome.Messages {
			p.sendToClient(payload)
		}
	}
}

func (p *PlayerActor) handleClientPayload(ctx actor.Context, m *MsgClientPayload) {
	p.conn = m.Conn
	model := p.domain.Model()
	model.IsOnline = true

	var b baseMsg
	_ = json.Unmarshal(m.Raw, &b)

	switch b.Type {
	case "C_Login":
		p.executeCommand(ctx, domain.NewLoginCommand())

	case "C_StartSeq":
		var req reqStart
		_ = json.Unmarshal(m.Raw, &req)
		p.executeCommand(ctx, domain.NewStartSequenceCommand(req.SeqID, req.SubProjectID))

	case "C_StopSeq":
		p.executeCommand(ctx, domain.NewStopSequenceCommand())

	case "C_ListBag":
		p.executeCommand(ctx, domain.NewListBagCommand())

	case "C_ListEquipment":
		p.executeCommand(ctx, domain.NewListEquipmentCommand(false))

	case "C_EquipItem":
		var req struct {
			ItemID      string `json:"item_id"`
			Enhancement int    `json:"enhancement"`
		}
		_ = json.Unmarshal(m.Raw, &req)
		p.executeCommand(ctx, domain.NewEquipItemCommand(req.ItemID, req.Enhancement))

	case "C_UnequipItem":
		var req struct {
			Slot string `json:"slot"`
		}
		_ = json.Unmarshal(m.Raw, &req)
		p.executeCommand(ctx, domain.NewUnequipItemCommand(domain.EquipmentSlot(req.Slot)))

	case "C_UseItem":
		var req struct {
			ItemID string `json:"item_id"`
			Count  int64  `json:"count"`
		}
		_ = json.Unmarshal(m.Raw, &req)
		p.executeCommand(ctx, domain.NewUseItemCommand(req.ItemID, req.Count))

	case "C_RemoveItem":
		var req struct {
			ItemID string `json:"item_id"`
			Count  int64  `json:"count"`
		}
		_ = json.Unmarshal(m.Raw, &req)
		p.executeCommand(ctx, domain.NewRemoveItemCommand(req.ItemID, req.Count))
	}
}

func (p *PlayerActor) executeCommand(ctx actor.Context, cmd domain.PlayerCommand) {
	if cmd == nil {
		return
	}
	result, err := p.commands.Handle(cmd)
	if err != nil {
		if ce, ok := err.(*domain.CommandError); ok {
			p.sendToClient(map[string]any{"type": "S_Error", "msg": ce.Message})
		} else {
			logx.Error("command execution failed", "player", p.playerID, "err", err)
			p.sendToClient(map[string]any{"type": "S_Error", "msg": "internal error"})
		}
		p.domain.Model().LastActive = time.Now()
		return
	}
	p.applyCommandResult(ctx, result)
}

func (p *PlayerActor) applyCommandResult(ctx actor.Context, result *domain.CommandResult) {
	if result == nil {
		return
	}
	if result.StopCurrent && p.currentSeq != nil {
		ctx.Stop(p.currentSeq)
		p.currentSeq = nil
	}
	if result.StartPlan != nil {
		pid := ctx.Spawn(actor.PropsFromProducer(func() actor.Actor {
			plan := result.StartPlan
			return NewSequenceActor(p.playerID, plan.SeqID, plan.Level, ctx.Self(), p.schedulerPID, plan.SubProject, plan.EquipmentBonus)
		}))
		ctx.Watch(pid)
		p.currentSeq = pid
	}
	for _, payload := range result.Responses {
		p.sendToClient(payload)
	}
	if result.PushEquipmentBonus && p.currentSeq != nil {
		ctx.Send(p.currentSeq, &MsgUpdateEquipmentBonus{Bonus: result.EquipmentBonus})
	}
	if result.Persist {
		p.saveState(ctx)
	}
}

func (p *PlayerActor) handleSeqResult(ctx actor.Context, m *SeqResult) {
	logx.Info("Player received SeqResult", "playerID", p.playerID, "seqID", m.SeqID, "gains", m.Gains, "items", len(m.Items), "isOnline", p.domain.Model().IsOnline)

	outcome, err := p.domain.ApplySequenceResult(&domain.SequenceResultData{
		Gains:        m.Gains,
		Rare:         m.Rare,
		Items:        m.Items,
		SeqID:        m.SeqID,
		Level:        m.Level,
		CurExp:       m.CurExp,
		Leveled:      m.Leveled,
		SubProjectID: m.SubProjectID,
	})
	if err != nil {
		logx.Error("failed to apply sequence result", "player", p.playerID, "err", err)
		return
	}
	for _, invErr := range outcome.InventoryErrors {
		logx.Error("Failed to add item", "player", p.playerID, "error", invErr)
	}
	for _, payload := range outcome.Messages {
		p.sendToClient(payload)
	}
	if outcome.ShouldPersist {
		p.saveState(ctx)
	}
}

func (p *PlayerActor) saveState(ctx actor.Context) {
	model := p.domain.Model()
	ctx.Send(p.persistPID, &MsgSavePlayer{
		PlayerID:          p.playerID,
		SeqLevels:         model.SeqLevels,
		Inventory:         model.Inventory,
		Exp:               model.Exp,
		Equipment:         model.Equipment.ExportState(),
		OfflineLimitHours: int64(model.OfflineLimit / time.Hour),
	})
}

func (p *PlayerActor) send(v any) {
	if p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}

func (p *PlayerActor) sendToClient(v any) {
	if p.domain.Model().IsOnline && p.conn != nil {
		_ = p.conn.WriteJSON(v)
	}
}
