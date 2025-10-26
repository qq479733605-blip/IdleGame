package player

import (
	"encoding/json"
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type MsgAttachConn struct {
	Conn         *websocket.Conn
	RequestState bool
}

type MsgDetachConn struct{}

type PlayerActor struct {
	playerID string
	services *Services
	state    *State
	domain   *Domain

	conn            *websocket.Conn
	currentSeq      *actor.PID
	currentSeqID    string
	commandRegistry map[string]CommandFactory
}

func NewPlayerActor(playerID string, services *Services) actor.Actor {
	state := NewState(playerID)
	domain := NewDomain(state)
	return &PlayerActor{
		playerID: playerID,
		services: services,
		state:    state,
		domain:   domain,
		commandRegistry: map[string]CommandFactory{
			common.CommandStartSequence: NewStartSequenceCommand,
			common.CommandStopSequence:  NewStopSequenceCommand,
			common.CommandUseItem:       NewUseItemCommand,
			common.CommandEquipItem:     NewEquipItemCommand,
			common.CommandUnequipItem:   NewUnequipItemCommand,
		},
	}
}

func (p *PlayerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		p.services.Register(ctx, p.playerID)
		p.services.RequestLoad(ctx, p.playerID)
		p.state.LastActive = time.Now()
	case *MsgAttachConn:
		p.handleAttachConn(msg)
	case *MsgDetachConn:
		p.conn = nil
		p.state.IsOnline = false
	case *common.MsgLoadResult:
		p.handleLoadResult(msg)
	case *common.MsgClientPayload:
		p.handleClientPayload(ctx, msg)
	case *common.MsgSequenceResult:
		p.handleSequenceResult(ctx, msg)
	case *common.MsgPlayerOffline:
		p.state.SetOnline(false)
		logx.Info("player offline", "player", p.playerID)
	case *common.MsgPlayerReconnect:
		if conn, ok := msg.Conn.(*websocket.Conn); ok {
			p.conn = conn
		}
		p.state.SetOnline(true)
		p.sendToClient(map[string]any{"type": "S_ReconnectOK"})
	case *common.MsgCheckExpire:
		if !p.state.IsOnline && !p.state.OfflineStart.IsZero() && time.Since(p.state.OfflineStart) > p.state.OfflineLimit {
			p.services.SaveSnapshot(ctx, p.state.Snapshot())
			p.services.Unregister(ctx, p.playerID)
			logx.Warn("player session expired", "player", p.playerID)
			ctx.Stop(ctx.Self())
		}
	case *common.MsgConnClosed:
		if conn, ok := msg.Conn.(*websocket.Conn); ok {
			if conn == p.conn {
				p.conn = nil
				p.state.SetOnline(false)
			}
		}
	case *actor.Terminated:
		if p.currentSeq != nil && msg.Who.Equal(p.currentSeq) {
			p.currentSeq = nil
			p.currentSeqID = ""
			p.state.ActiveSubProject = ""
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

func (p *PlayerActor) handleAttachConn(msg *MsgAttachConn) {
	p.conn = msg.Conn
	p.state.SetOnline(true)
	p.state.LastActive = time.Now()

	if msg.RequestState {
		reward := p.domain.OfflineRewards()
		if reward.Gains > 0 || len(reward.Items) > 0 {
			p.state.Exp += reward.Gains
			for itemID, count := range reward.Items {
				_ = p.state.Inventory.AddItem(common.ItemDrop{ID: itemID, Name: itemID}, count)
			}
			p.sendToClient(map[string]any{
				"type":             "S_OfflineReward",
				"gains":            reward.Gains,
				"offline_duration": int64(reward.Duration.Seconds()),
				"offline_items":    reward.Items,
				"bag":              p.state.Inventory.List(),
			})
		}
		payload := p.domain.BuildReconnectPayload(p.currentSeqID, p.getCurrentSeqLevel(), p.currentSeq != nil)
		p.sendToClient(payload)
	} else {
		p.sendToClient(map[string]any{"type": "S_Reconnected", "msg": "重连成功"})
	}
	p.state.OfflineStart = time.Time{}
}

func (p *PlayerActor) handleLoadResult(msg *common.MsgLoadResult) {
	if msg.Err != nil || msg.Snapshot == nil {
		p.domain.ApplySnapshot(nil)
		p.sendToClient(p.domain.PrepareNewPlayerResponse())
		return
	}
	p.domain.ApplySnapshot(msg.Snapshot)
	p.sendToClient(p.domain.PrepareLoadResponse())
}

func (p *PlayerActor) handleClientPayload(ctx actor.Context, msg *common.MsgClientPayload) {
	if conn, ok := msg.Conn.(*websocket.Conn); ok {
		p.conn = conn
	}
	p.state.SetOnline(true)
	p.state.LastActive = time.Now()

	var base struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(msg.Raw, &base)
	switch base.Type {
	case "C_Login":
		p.sendToClient(map[string]any{
			"type":               "S_LoginOK",
			"msg":                "登录成功",
			"playerId":           p.playerID,
			"exp":                p.state.Exp,
			"seq_levels":         p.state.SeqLevels,
			"bag":                p.state.Inventory.List(),
			"equipment":          p.state.Equipment.ExportView(),
			"equipment_bonus":    p.state.Equipment.TotalBonus(),
			"is_running":         p.currentSeq != nil,
			"seq_id":             p.currentSeqID,
			"seq_level":          p.getCurrentSeqLevel(),
			"active_sub_project": p.state.ActiveSubProject,
		})
		return
	case common.CommandListBag:
		p.sendToClient(map[string]any{"type": "S_BagInfo", "bag": p.state.Inventory.List()})
		return
	case common.CommandListEquipment:
		p.sendEquipmentState(false)
		return
	}

	factory, ok := p.commandRegistry[base.Type]
	if !ok {
		logx.Warn("unknown command", "type", base.Type)
		return
	}
	cmd := factory(msg.Raw)
	_ = cmd.Execute(&CommandContext{ActorCtx: ctx, Domain: p.domain, Services: p.services, Player: p, RawPayload: msg.Raw})
}

func (p *PlayerActor) handleSequenceResult(ctx actor.Context, msg *common.MsgSequenceResult) {
	payload := p.domain.ApplySequenceResult(msg)
	if p.conn != nil {
		p.sendToClient(map[string]any{
			"type":            "S_SeqResult",
			"gains":           payload.Gains,
			"rare":            payload.Rare,
			"bag":             payload.Bag,
			"seq_id":          payload.SeqID,
			"level":           payload.Level,
			"cur_exp":         payload.CurExp,
			"leveled":         payload.Leveled,
			"items":           payload.Items,
			"sub_project_id":  payload.SubProjectID,
			"equipment_bonus": payload.Bonus,
		})
	}
	p.services.SaveSnapshot(ctx, p.state.Snapshot())
}

func (p *PlayerActor) pushEquipmentBonus(ctx actor.Context) {
	if p.currentSeq != nil {
		ctx.Send(p.currentSeq, &common.MsgUpdateEquipmentBonus{Bonus: p.state.Equipment.TotalBonus()})
	}
	p.sendEquipmentState(false)
}

func (p *PlayerActor) sendEquipmentState(includeCatalog bool) {
	payload := map[string]any{
		"type":               "S_EquipmentState",
		"equipment":          p.state.Equipment.ExportView(),
		"equipment_bonus":    p.state.Equipment.TotalBonus(),
		"active_sub_project": p.state.ActiveSubProject,
	}
	if includeCatalog {
		payload["catalog"] = GetEquipmentCatalogSummary()
	}
	p.sendToClient(payload)
}

func (p *PlayerActor) sendToClient(payload any) {
	if p.conn == nil {
		return
	}
	if err := p.conn.WriteJSON(payload); err != nil {
		logx.Warn("send ws", "player", p.playerID, "err", err)
	}
}

func (p *PlayerActor) sendError(msg string) {
	p.sendToClient(map[string]any{"type": "S_Error", "msg": msg})
}

func (p *PlayerActor) getCurrentSeqLevel() int {
	if p.currentSeqID == "" {
		return 0
	}
	return p.state.SeqLevels[p.currentSeqID]
}
