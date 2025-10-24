package actors

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// 预留的集中调度器：当前骨架未启用。
// 未来可用它来统一广播 SeqTick，避免每个 Sequence 自己计时。

type SchedulerActor struct{}

type AddTarget struct{ PID *actor.PID }

type RemoveTarget struct{ PID *actor.PID }

func (s *SchedulerActor) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case *actor.Started:
		_ = time.Second // placeholder，避免未使用导入
	}
}
