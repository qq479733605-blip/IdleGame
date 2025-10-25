package actors

import (
	"time"

	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
)

type SequenceActor struct {
	playerID       string
	seq            *domain.Sequence
	cfg            *domain.SequenceConfig
	parent         *actor.PID
	scheduler      *actor.PID
	tickInterval   time.Duration
	equipmentBonus domain.EquipmentBonus
}

func NewSequenceActor(playerID, seqID string, level int, parent, scheduler *actor.PID, subProject *domain.SequenceSubProject, bonus domain.EquipmentBonus) actor.Actor {
	cfg, ok := domain.GetSequenceConfig(seqID)
	if !ok {
		panic("sequence config not found: " + seqID)
	}
	seq := &domain.Sequence{
		ID:        seqID,
		Level:     level,
		StartTime: time.Now(),
		LastTick:  time.Now(),
	}
	if subProject != nil {
		seq.SetSubProject(subProject)
	}
	interval := cfg.EffectiveInterval(subProject)
	if interval <= 0 {
		interval = time.Duration(cfg.TickInterval) * time.Second
	}
	return &SequenceActor{
		playerID:       playerID,
		seq:            seq,
		cfg:            cfg,
		parent:         parent,
		scheduler:      scheduler,
		tickInterval:   interval,
		equipmentBonus: bonus,
	}
}

func (s *SequenceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		if s.scheduler != nil {
			ctx.Send(s.scheduler, &AddTarget{PID: ctx.Self(), Interval: s.tickInterval})
		} else {
			s.scheduleNext(ctx)
		}

	case *SeqTick:
		r := s.seq.Tick(s.cfg, s.equipmentBonus)
		var rareName string
		if r.RareEvt != nil {
			rareName = r.RareEvt.Name
		}

		logx.Info("Sequence Tick", "playerID", s.playerID, "seqID", s.seq.ID,
			"gains", r.Gains, "items", len(r.Items), "rareEvent", rareName)

		for _, item := range r.Items {
			logx.Info("Item drop", "itemID", item.ID, "itemName", item.Name, "chance", item.DropChance)
		}

		var rareEvents []string
		if r.RareEvt != nil {
			rareEvents = []string{rareName}
		}

		ctx.Send(s.parent, &SeqResult{
			Gains:        r.Gains,
			Rare:         rareEvents,
			Items:        r.Items,
			SeqID:        s.seq.ID,
			Level:        r.Level,
			CurExp:       r.CurExp,
			Leveled:      r.Leveled,
			SubProjectID: r.SubProjectID,
		})
		if s.scheduler == nil {
			s.scheduleNext(ctx)
		}

	case *MsgUpdateEquipmentBonus:
		s.equipmentBonus = msg.Bonus

	case *SeqStop:
		s.unregister(ctx)
		ctx.Stop(ctx.Self())

	case *actor.Stopping:
		s.unregister(ctx)
	}
}

func (s *SequenceActor) scheduleNext(ctx actor.Context) {
	interval := s.tickInterval
	if interval <= 0 {
		interval = time.Duration(s.cfg.TickInterval) * time.Second
	}
	time.AfterFunc(interval, func() {
		ctx.Send(ctx.Self(), &SeqTick{})
	})
}

func (s *SequenceActor) unregister(ctx actor.Context) {
	if s.scheduler != nil {
		ctx.Send(s.scheduler, &RemoveTarget{PID: ctx.Self()})
	}
}
