package actors

import (
	"fmt"
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

type SeqTick struct{}

func NewSequenceActor(playerID, seqID string, level int, parent *actor.PID) actor.Actor {
	cfg, ok := domain.GetSequenceConfig(seqID)
	if !ok {
		panic(fmt.Sprintf("sequence config not found: %s", seqID))
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
		result := s.seq.Tick(s.cfg)

		var rareName string
		if result.RareEvt != nil {
			rareName = result.RareEvt.Name
		}

		// ⚡ 向父 Actor 汇报完整掉落
		ctx.Send(s.parent, &SeqResult{
			Gains: result.Gains,
			Rare:  []string{rareName},
			Items: result.Items, // 新增字段：掉落物品
		})

		s.scheduleNext(ctx)

	case *SeqStop:
		ctx.Stop(ctx.Self())
	}
}

func (s *SequenceActor) scheduleNext(ctx actor.Context) {
	time.AfterFunc(time.Duration(s.cfg.TickInterval)*time.Second, func() {
		ctx.Send(ctx.Self(), &SeqTick{})
	})
}
