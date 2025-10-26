package game

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"idlemmoserver/common"
	"idlemmoserver/game/internal/game/domain"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/nats-io/nats.go"
)

type clientCommand struct {
	Raw json.RawMessage
}

type PlayerActor struct {
	playerID string
	nc       *nats.Conn
	snapshot common.PlayerSnapshot
}

func NewPlayerActor(playerID string, nc *nats.Conn) actor.Actor {
	return &PlayerActor{
		playerID: playerID,
		nc:       nc,
		snapshot: domain.NewPlayerSnapshot(playerID),
	}
}

func (p *PlayerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		p.loadSnapshot()
	case *clientCommand:
		p.handleClientCommand(msg.Raw)
	case *actor.Stopping:
		p.saveSnapshot()
	}
}

func (p *PlayerActor) loadSnapshot() {
	req := common.PersistLoadRequest{PlayerID: p.playerID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	msg, err := p.nc.RequestWithContext(ctx, common.SubjectPersistLoad, common.MustMarshal(req))
	if err != nil {
		log.Printf("player %s load snapshot: %v", p.playerID, err)
		return
	}
	var resp common.PersistLoadResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		log.Printf("player %s parse snapshot: %v", p.playerID, err)
		return
	}
	if resp.Found {
		p.snapshot = resp.Snapshot
	}
}

func (p *PlayerActor) saveSnapshot() {
	req := common.PersistSaveRequest{Snapshot: p.snapshot}
	if err := p.nc.Publish(common.SubjectPersistSave, common.MustMarshal(req)); err != nil {
		log.Printf("player %s save snapshot: %v", p.playerID, err)
	}
}

func (p *PlayerActor) handleClientCommand(raw json.RawMessage) {
	var msg common.ClientMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	switch msg.Type {
	case "C_Ping":
		p.pushToClient(map[string]string{"type": "S_Pong"})
	case "C_AddExp":
		var payload struct {
			Amount int64 `json:"amount"`
		}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		if payload.Amount <= 0 {
			return
		}
		p.snapshot.Exp += payload.Amount
		p.saveSnapshot()
		p.pushToClient(map[string]interface{}{
			"type": "S_ExpUpdate",
			"exp":  p.snapshot.Exp,
		})
	case "C_GetState":
		p.pushToClient(map[string]interface{}{
			"type":     "S_State",
			"snapshot": p.snapshot,
		})
	}
}

func (p *PlayerActor) pushToClient(msg interface{}) {
	payload := common.GameToGateMessage{PlayerID: p.playerID, Message: msg}
	if err := p.nc.Publish(common.SubjectGameToGate, common.MustMarshal(payload)); err != nil {
		log.Printf("player %s publish to gate: %v", p.playerID, err)
	}
}
