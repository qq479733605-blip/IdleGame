package persist

import (
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
)

type Actor struct {
	repo   PlayerRepository
	online map[string]*actor.PID
	ticker *time.Ticker
}

func NewActor(repo PlayerRepository) *Actor {
	return &Actor{repo: repo, online: make(map[string]*actor.PID)}
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.ticker = time.NewTicker(60 * time.Second)
		go func() {
			for range a.ticker.C {
				for _, pid := range a.online {
					ctx.Send(pid, &common.MsgCheckExpire{})
				}
			}
		}()
	case *common.MsgRegisterPlayer:
		a.online[msg.PlayerID] = msg.PID
	case *common.MsgUnregisterPlayer:
		delete(a.online, msg.PlayerID)
	case *common.MsgSavePlayer:
		if err := a.repo.SavePlayer(&msg.Snapshot); err != nil {
			logx.Error("persist save failed", "err", err.Error())
		}
	case *common.MsgLoadPlayer:
		data, err := a.repo.LoadPlayer(msg.PlayerID)
		ctx.Send(msg.ReplyTo, &common.MsgLoadResult{Snapshot: data, Err: err})
	case *actor.Stopping:
		if a.ticker != nil {
			a.ticker.Stop()
		}
	}
}
