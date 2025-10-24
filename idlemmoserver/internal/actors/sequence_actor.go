package actors

import (
	"time"

	"idlemmoserver/internal/domain"

	"github.com/asynkron/protoactor-go/actor"
)

type SequenceActor struct {
	playerID string
	seq      *domain.Sequence
	cfg      *domain.SequenceConfig
	parent   *actor.PID
}

func NewSequenceActor(playerID, seqID string, level int, parent *actor.PID) actor.Actor {
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
	return &SequenceActor{
		playerID: playerID,
		seq:      seq,
		cfg:      cfg,
		parent:   parent,
	}
}

func (s *SequenceActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *actor.Started:
		s.scheduleNext(ctx)

	case *SeqTick:
		r := s.seq.Tick(s.cfg)
		var rareName string
		if r.RareEvt != nil {
			rareName = r.RareEvt.Name
		}
		ctx.Send(s.parent, &SeqResult{
			Gains:   r.Gains,
			Rare:    []string{rareName},
			Items:   r.Items,
			SeqID:   s.seq.ID,
			Level:   r.Level,
			CurExp:  r.CurExp,
			Leveled: r.Leveled,
		})
		s.scheduleNext(ctx)

	case *SeqStop, *actor.Stopping:
		ctx.Stop(ctx.Self())
	}
}

func (s *SequenceActor) scheduleNext(ctx actor.Context) {
	time.AfterFunc(time.Duration(s.cfg.TickInterval)*time.Second, func() {
		ctx.Send(ctx.Self(), &SeqTick{})
	})
}
