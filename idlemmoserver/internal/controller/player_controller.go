package controller

import (
	"encoding/json"
	"errors"
	"time"

	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/service"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type baseMsg struct {
	Type string `json:"type"`
}

type reqStart struct {
	Type         string `json:"type"`
	SeqID        string `json:"seq_id"`
	Target       int64  `json:"target"`
	SubProjectID string `json:"sub_project_id"`
}

// PlayerRuntime describes the actor-level operations exposed to the controller layer.
type PlayerRuntime interface {
	CurrentSequence() *actor.PID
	StartSequence(ctx actor.Context, seqID string, level int, subProject *domain.SequenceSubProject, bonus domain.EquipmentBonus) (*actor.PID, error)
	StopCurrentSequence(ctx actor.Context) bool
	PushEquipmentBonus(ctx actor.Context)
	PersistPlayer(ctx actor.Context)
}

// PlayerController owns request parsing and delegates gameplay logic to services.
type PlayerController struct {
	model            *domain.PlayerModel
	runtime          PlayerRuntime
	sequenceService  *service.SequenceService
	itemService      *service.ItemService
	equipmentService *service.EquipmentService
	conn             *websocket.Conn
}

func NewPlayerController(model *domain.PlayerModel, runtime PlayerRuntime) *PlayerController {
	return &PlayerController{
		model:            model,
		runtime:          runtime,
		sequenceService:  service.NewSequenceService(model),
		itemService:      service.NewItemService(model),
		equipmentService: service.NewEquipmentService(model),
	}
}

func (c *PlayerController) AttachConn(ctx actor.Context, conn *websocket.Conn, requestState bool) {
	c.conn = conn
	c.model.IsOnline = true
	c.model.LastActive = time.Now()

	logx.Info("收到 MsgAttachConn", "playerID", c.model.PlayerID, "requestState", requestState)

	if requestState {
		offlineGain, offlineItems, duration := c.sequenceService.CalculateOfflineRewards()
		if offlineGain > 0 || len(offlineItems) > 0 {
			c.sequenceService.ApplyOfflineRewards(offlineGain, offlineItems)
			c.sendToClient(map[string]any{
				"type":             "S_OfflineReward",
				"gains":            offlineGain,
				"offline_duration": int64(duration.Seconds()),
				"offline_items":    offlineItems,
				"bag":              c.model.Inventory.List(),
			})
		}

		payload := c.sequenceService.BuildReconnectedPayload()
		logx.Info("发送 S_Reconnected 消息", "playerID", c.model.PlayerID, "payload", payload)
		c.sendToClient(payload)
	} else {
		c.sendToClient(map[string]any{"type": "S_Reconnected", "msg": "重连成功"})
	}

	c.model.OfflineStart = time.Time{}
}

func (c *PlayerController) DetachConn() {
	if c.conn != nil {
		c.conn = nil
		logx.Info("🕓 Player %s disconnected (actor retained)", c.model.PlayerID)
	}
}

func (c *PlayerController) HandleClientPayload(ctx actor.Context, conn *websocket.Conn, raw []byte) {
	c.conn = conn
	c.model.IsOnline = true
	c.model.LastActive = time.Now()

	var b baseMsg
	_ = json.Unmarshal(raw, &b)

	switch b.Type {
	case "C_Login":
		c.sendToClient(map[string]any{
			"type":     "S_LoginOK",
			"msg":      "登录成功",
			"playerId": c.model.PlayerID,
			"exp":      c.model.Exp,
		})

	case "C_StartSeq":
		c.handleStartSequence(ctx, raw)

	case "C_StopSeq":
		if c.runtime.StopCurrentSequence(ctx) {
			c.sequenceService.OnSequenceStopped()
			c.sendToClient(map[string]any{"type": "S_SeqEnded"})
		}

	case "C_ListBag":
		c.sendToClient(map[string]any{"type": "S_BagInfo", "bag": c.model.Inventory.List()})

	case "C_ListEquipment":
		c.sendToClient(c.equipmentService.EquipmentState(false))

	case "C_EquipItem":
		c.handleEquipItem(ctx, raw)

	case "C_UnequipItem":
		c.handleUnequipItem(ctx, raw)

	case "C_UseItem":
		c.handleUseItem(raw)

	case "C_RemoveItem":
		c.handleRemoveItem(raw)
	}
}

func (c *PlayerController) handleStartSequence(ctx actor.Context, raw []byte) {
	var req reqStart
	_ = json.Unmarshal(raw, &req)

	if c.runtime.CurrentSequence() != nil {
		c.runtime.StopCurrentSequence(ctx)
		c.sequenceService.OnSequenceStopped()
	}

	cfg, subProject, level, err := c.sequenceService.PrepareStartSequence(req.SeqID, req.SubProjectID)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	bonus := c.model.Equipment.TotalBonus()
	if _, err := c.runtime.StartSequence(ctx, req.SeqID, level, subProject, bonus); err != nil {
		c.sendError(err.Error())
		return
	}

	subProjectID := ""
	if subProject != nil {
		subProjectID = subProject.ID
	}
	c.sequenceService.OnSequenceStarted(req.SeqID, subProjectID)

	interval := cfg.EffectiveInterval(subProject)
	c.sendToClient(map[string]any{
		"type":            "S_SeqStarted",
		"seq_id":          req.SeqID,
		"level":           level,
		"sub_project_id":  subProjectID,
		"tick_interval":   interval.Seconds(),
		"equipment_bonus": bonus,
	})
}

func (c *PlayerController) handleEquipItem(ctx actor.Context, raw []byte) {
	var req struct {
		ItemID      string `json:"item_id"`
		Enhancement int    `json:"enhancement"`
	}
	_ = json.Unmarshal(raw, &req)

	if err := c.equipmentService.EquipItem(req.ItemID, req.Enhancement); err != nil {
		c.sendError(err.Error())
		return
	}

	c.runtime.PushEquipmentBonus(ctx)
	c.sendToClient(c.equipmentService.EquipmentChangedPayload())
}

func (c *PlayerController) handleUnequipItem(ctx actor.Context, raw []byte) {
	var req struct {
		Slot string `json:"slot"`
	}
	_ = json.Unmarshal(raw, &req)

	if req.Slot == "" {
		c.sendError("请选择要卸下的位置")
		return
	}

	slot := domain.EquipmentSlot(req.Slot)
	if err := c.equipmentService.UnequipItem(slot); err != nil {
		c.sendError(err.Error())
		return
	}

	c.runtime.PushEquipmentBonus(ctx)
	c.sendToClient(c.equipmentService.EquipmentChangedPayload())
}

func (c *PlayerController) handleUseItem(raw []byte) {
	var req struct {
		ItemID string `json:"item_id"`
		Count  int64  `json:"count"`
	}
	_ = json.Unmarshal(raw, &req)

	payload, err := c.itemService.UseItem(req.ItemID, req.Count)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	c.sendToClient(payload)
}

func (c *PlayerController) handleRemoveItem(raw []byte) {
	var req struct {
		ItemID string `json:"item_id"`
		Count  int64  `json:"count"`
	}
	_ = json.Unmarshal(raw, &req)

	if err := c.itemService.RemoveItem(req.ItemID, req.Count); err != nil {
		c.sendError(err.Error())
		return
	}
	c.sendToClient(map[string]any{"type": "S_ItemRemoved", "item_id": req.ItemID, "count": req.Count})
}

func (c *PlayerController) HandleSequenceResult(ctx actor.Context, result service.SequenceResult) {
	payload := c.sequenceService.ApplySequenceResult(result)
	if c.model.IsOnline {
		c.sendToClient(payload)
	}
	c.runtime.PersistPlayer(ctx)
}

func (c *PlayerController) HandlePlayerOffline() {
	c.model.IsOnline = false
	c.model.OfflineStart = time.Now()
	logx.Info("player offline", "player", c.model.PlayerID, "limit", c.model.OfflineLimit)
}

func (c *PlayerController) HandleReconnect(conn *websocket.Conn) {
	c.conn = conn
	c.model.IsOnline = true
	c.model.LastActive = time.Now()
	c.sendToClient(map[string]any{"type": "S_ReconnectOK"})
}

func (c *PlayerController) HandleConnClosed(conn *websocket.Conn) {
	if conn == c.conn {
		c.conn = nil
		c.model.IsOnline = false
		c.model.OfflineStart = time.Now()
	}
}

func (c *PlayerController) OnSequenceTerminated() {
	c.sequenceService.OnSequenceStopped()
	if c.model.IsOnline {
		c.sendToClient(map[string]any{"type": "S_SeqEnded"})
	}
}

func (c *PlayerController) SendNewPlayer() {
	c.sendToClient(map[string]any{"type": "S_NewPlayer"})
}

func (c *PlayerController) SendLoadOK(offlineLimitHours int64) {
	c.sendToClient(map[string]any{
		"type":                "S_LoadOK",
		"exp":                 c.model.Exp,
		"bag":                 c.model.Inventory.List(),
		"offline_limit_hours": offlineLimitHours,
		"equipment":           c.model.Equipment.Export(),
		"equipment_bonus":     c.model.Equipment.TotalBonus(),
	})
}

func (c *PlayerController) sendError(msg string) {
	if msg == "" {
		msg = errors.New("unknown error").Error()
	}
	c.sendToClient(map[string]any{"type": "S_Error", "msg": msg})
}

func (c *PlayerController) sendToClient(v any) {
	if c.model.IsOnline && c.conn != nil {
		_ = c.conn.WriteJSON(v)
	}
}
