package sequence

import (
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/scheduler"

	"github.com/asynkron/protoactor-go/actor"
)

type Actor struct {
	playerID       string
	seq            *Sequence
	cfg            *Config
	parent         *actor.PID
	scheduler      *actor.PID
	tickInterval   time.Duration
	equipmentBonus common.EquipmentBonus
}

type Params struct {
	PlayerID  string
	SeqID     string
	Level     int
	Sub       *SubProject
	Parent    *actor.PID
	Scheduler *actor.PID
	Bonus     common.EquipmentBonus
}

func NewActor(params Params) actor.Actor {
	cfg, ok := GetConfig(params.SeqID)
	if !ok {
		panic("sequence config not found: " + params.SeqID)
	}
	seq := NewSequence(params.SeqID, params.Level, params.Sub)
	interval := seq.EffectiveInterval(cfg)
	if interval <= 0 {
		interval = time.Duration(cfg.TickInterval) * time.Second
	}
	return &Actor{
		playerID:       params.PlayerID,
		seq:            seq,
		cfg:            cfg,
		parent:         params.Parent,
		scheduler:      params.Scheduler,
		tickInterval:   interval,
		equipmentBonus: params.Bonus,
	}
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		if a.scheduler != nil {
			ctx.Send(a.scheduler, &scheduler.AddTarget{PID: ctx.Self(), Interval: a.tickInterval})
		} else {
			a.scheduleNext(ctx)
		}
	case *common.MsgSequenceTick:
		result := a.seq.Tick(a.cfg, a.equipmentBonus)
		rareNames := []string{}
		if result.RareEvt != nil {
			rareNames = []string{result.RareEvt.Name}
		}
		ctx.Send(a.parent, &common.MsgSequenceResult{
			PlayerID:     a.playerID,
			SeqID:        a.seq.ID,
			Items:        result.Items,
			Rare:         rareNames,
			Gains:        result.Gains,
			Level:        result.Level,
			CurExp:       result.CurExp,
			Leveled:      result.Leveled,
			SubProjectID: result.SubProjectID,
		})
		if a.scheduler == nil {
			a.scheduleNext(ctx)
		}
	case *common.MsgUpdateEquipmentBonus:
		a.equipmentBonus = msg.Bonus
	case *common.MsgSequenceStop:
		a.unregister(ctx)
		ctx.Stop(ctx.Self())
	case *actor.Terminated:
		a.unregister(ctx)
	}
}

func (a *Actor) scheduleNext(ctx actor.Context) {
	interval := a.tickInterval
	if interval <= 0 {
		interval = time.Duration(a.cfg.TickInterval) * time.Second
	}
	time.AfterFunc(interval, func() {
		ctx.Send(ctx.Self(), &common.MsgSequenceTick{})
	})
}

func (a *Actor) unregister(ctx actor.Context) {
	if a.scheduler != nil {
		ctx.Send(a.scheduler, &scheduler.RemoveTarget{PID: ctx.Self()})
	}
}
