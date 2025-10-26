package player

import (
	"idlemmoserver/internal/common"
	"idlemmoserver/internal/sequence"

	"github.com/asynkron/protoactor-go/actor"
)

type Services struct {
	PersistPID   *actor.PID
	SchedulerPID *actor.PID
	GatewayPID   *actor.PID
}

func (s *Services) SaveSnapshot(ctx actor.Context, snapshot common.PlayerSnapshot) {
	if s.PersistPID == nil {
		return
	}
	ctx.Send(s.PersistPID, &common.MsgSavePlayer{Snapshot: snapshot})
}

func (s *Services) Register(ctx actor.Context, playerID string) {
	if s.PersistPID == nil {
		return
	}
	ctx.Send(s.PersistPID, &common.MsgRegisterPlayer{PlayerID: playerID, PID: ctx.Self()})
}

func (s *Services) Unregister(ctx actor.Context, playerID string) {
	if s.PersistPID == nil {
		return
	}
	ctx.Send(s.PersistPID, &common.MsgUnregisterPlayer{PlayerID: playerID})
	if s.GatewayPID != nil {
		ctx.Send(s.GatewayPID, &common.MsgUnregisterPlayer{PlayerID: playerID})
	}
}

func (s *Services) RequestLoad(ctx actor.Context, playerID string) {
	if s.PersistPID == nil {
		return
	}
	ctx.Send(s.PersistPID, &common.MsgLoadPlayer{PlayerID: playerID, ReplyTo: ctx.Self()})
}

func (s *Services) SpawnSequence(ctx actor.Context, params sequence.Params) *actor.PID {
	props := actor.PropsFromProducer(func() actor.Actor {
		return sequence.NewActor(params)
	})
	pid := ctx.Spawn(props)
	ctx.Watch(pid)
	return pid
}

func (s *Services) StopSequence(ctx actor.Context, pid *actor.PID) {
	if pid == nil {
		return
	}
	ctx.Unwatch(pid)
	ctx.Stop(pid)
}
