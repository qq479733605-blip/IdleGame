package scheduler

import (
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
)

type Actor struct {
	targets      map[string]*target
	tickInterval time.Duration
}

type target struct {
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

func NewActor(interval time.Duration) *Actor {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	return &Actor{
		targets:      make(map[string]*target),
		tickInterval: interval,
	}
}

func (a *Actor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		logx.Info("SchedulerActor started", "interval", a.tickInterval.String())
		a.scheduleNext(ctx)
	case *AddTarget:
		interval := msg.Interval
		if interval <= 0 {
			interval = time.Second
		}
		key := msg.PID.String()
		a.targets[key] = &target{PID: msg.PID, Interval: interval, NextTick: time.Now().Add(interval)}
		logx.Info("scheduler add target", "pid", key, "interval", interval.String())
	case *RemoveTarget:
		if msg.PID != nil {
			delete(a.targets, msg.PID.String())
			logx.Info("scheduler remove target", "pid", msg.PID.String())
		}
	case *schedulerTick:
		now := time.Now()
		for key, tgt := range a.targets {
			if now.After(tgt.NextTick) || now.Equal(tgt.NextTick) {
				ctx.Send(tgt.PID, &common.MsgSequenceTick{})
				tgt.NextTick = now.Add(tgt.Interval)
				logx.Info("scheduler tick", "pid", key)
			}
		}
		a.scheduleNext(ctx)
	case *actor.Terminated:
		delete(a.targets, msg.Who.String())
	case *actor.Stopping:
		a.targets = make(map[string]*target)
	}
}

func (a *Actor) scheduleNext(ctx actor.Context) {
	time.AfterFunc(a.tickInterval, func() {
		ctx.Send(ctx.Self(), &schedulerTick{})
	})
}
