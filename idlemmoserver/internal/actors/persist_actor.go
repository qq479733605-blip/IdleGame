package actors

import (
	"idlemmoserver/internal/logx"
	"time"

	"idlemmoserver/internal/persist"

	"github.com/asynkron/protoactor-go/actor"
)

type PersistActor struct {
	repo   persist.Repository
	online map[string]*actor.PID // 已注册玩家
	ticker *time.Ticker
}

func NewPersistActor(repo persist.Repository) *PersistActor {
	return &PersistActor{repo: repo, online: make(map[string]*actor.PID)}
}

func (p *PersistActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *actor.Started:
		// 每 60 秒检查一次离线超时
		p.ticker = time.NewTicker(60 * time.Second)
		go func() {
			for range p.ticker.C {
				for _, pid := range p.online {
					ctx.Send(pid, &MsgCheckExpire{})
				}
			}
		}()

	case *MsgRegisterPlayer:
		p.online[m.PlayerID] = m.PID

	case *MsgUnregisterPlayer:
		delete(p.online, m.PlayerID)

	case *MsgSavePlayer:
		bag := m.Inventory.List()
		err := p.repo.SavePlayer(&persist.PlayerData{
			PlayerID:          m.PlayerID,
			SeqLevels:         m.SeqLevels,
			Inventory:         bag,
			Exp:               m.Exp,
			OfflineLimitHours: m.OfflineLimitHours,
		})
		if err != nil {
			logx.Error("persist save failed", "err", err.Error())
		}

	case *MsgLoadPlayer:
		data, err := p.repo.LoadPlayer(m.PlayerID)
		ctx.Send(m.ReplyTo, &MsgLoadResult{Data: data, Err: err})

	case *actor.Stopping:
		if p.ticker != nil {
			p.ticker.Stop()
		}
	}
}
