package actors

import (
	"time"

	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
)

type SchedulerActor struct {
	targets      map[string]*scheduleTarget
	tickInterval time.Duration
}

type scheduleTarget struct {
	PID      *actor.PID
	Interval time.Duration
	NextTick time.Time
}

type schedulerTick struct{}

type AddTarget struct {
	PID      *actor.PID
	Interval time.Duration
}

type RemoveTarget struct{ PID *actor.PID }

func NewSchedulerActor() *SchedulerActor {
	return &SchedulerActor{
		targets:      make(map[string]*scheduleTarget),
		tickInterval: 200 * time.Millisecond,
	}
}

func (s *SchedulerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logx.Info("SchedulerActor started", "interval", s.tickInterval.String())
		s.scheduleNext(ctx)
	case *AddTarget:
		interval := msg.Interval
		if interval <= 0 {
			interval = time.Second
		}
		key := msg.PID.String()
		s.targets[key] = &scheduleTarget{
			PID:      msg.PID,
			Interval: interval,
			NextTick: time.Now().Add(interval),
		}
		logx.Info("scheduler add target", "pid", key, "interval", interval.String())
	case *RemoveTarget:
		if msg.PID != nil {
			delete(s.targets, msg.PID.String())
			logx.Info("scheduler remove target", "pid", msg.PID.String())
		}
	case *schedulerTick:
		now := time.Now()
		for key, target := range s.targets {
			if now.After(target.NextTick) || now.Equal(target.NextTick) {
				ctx.Send(target.PID, &SeqTick{})
				target.NextTick = now.Add(target.Interval)
				logx.Info("scheduler tick", "pid", key)
			}
		}
		s.scheduleNext(ctx)
	case *actor.Terminated:
		delete(s.targets, msg.Who.String())
	case *actor.Stopping:
		s.targets = make(map[string]*scheduleTarget)
	}
}

func (s *SchedulerActor) scheduleNext(ctx actor.Context) {
	time.AfterFunc(s.tickInterval, func() {
		ctx.Send(ctx.Self(), &schedulerTick{})
	})
}
