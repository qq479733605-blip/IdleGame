package gateway

import (
	"sync"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/player"

	"github.com/asynkron/protoactor-go/actor"
)

type GatewayActor struct {
	root     *actor.RootContext
	services *player.Services
	players  sync.Map
}

func NewGatewayActor(root *actor.RootContext, services *player.Services) actor.Actor {
	return &GatewayActor{root: root, services: services}
}

func (g *GatewayActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logx.Info("GatewayActor started")
	case *common.MsgEnsurePlayer:
		pid := g.ensurePlayer(ctx, msg.PlayerID)
		if msg.ReplyTo != nil {
			ctx.Respond(&common.MsgPlayerReady{PlayerPID: pid})
		}
	case *common.MsgUnregisterPlayer:
		g.players.Delete(msg.PlayerID)
		removePlayerActor(msg.PlayerID)
	}
}

func (g *GatewayActor) ensurePlayer(ctx actor.Context, playerID string) *actor.PID {
	if val, ok := g.players.Load(playerID); ok {
		return val.(*actor.PID)
	}
	props := actor.PropsFromProducer(func() actor.Actor {
		return player.NewPlayerActor(playerID, g.services)
	})
	pid := ctx.Spawn(props)
	g.players.Store(playerID, pid)
	logx.Info("spawn player actor", "player", playerID, "pid", pid.String())
	return pid
}
