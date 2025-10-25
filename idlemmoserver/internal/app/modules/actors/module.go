package actorsmod

import (
	"context"

	"idlemmoserver/internal/actors"
	"idlemmoserver/internal/app/runtime"
	"idlemmoserver/internal/gateway"
	"idlemmoserver/internal/persist"

	"github.com/asynkron/protoactor-go/actor"
)

// Module provisions the actor system and long-lived gateway/persistence actors.
type Module struct {
	saveDir string
}

// New creates an actor module that persists data inside the provided directory.
func New(saveDir string) *Module {
	return &Module{saveDir: saveDir}
}

func (m *Module) Name() string { return "actors" }

func (m *Module) Configure(ctx context.Context, c *runtime.Container) error {
	sys := actor.NewActorSystem()
	root := sys.Root

	repo := persist.NewJSONRepo(m.saveDir)
	c.MustProvide(runtime.ServiceActorSystem, sys)
	c.MustProvide(runtime.ServiceActorRoot, root)
	c.MustProvide(runtime.ServiceGameRepository, repo)

	persistPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return actors.NewPersistActor(repo)
	}))
	schedulerPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return actors.NewSchedulerActor()
	}))
	gateway.SetPersistPID(persistPID)
	gateway.SetSchedulerPID(schedulerPID)

	gatewayPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return actors.NewGatewayActor(root, persistPID, schedulerPID)
	}))

	c.MustProvide(runtime.ServicePersistPID, persistPID)
	c.MustProvide(runtime.ServiceSchedulerPID, schedulerPID)
	c.MustProvide(runtime.ServiceGatewayPID, gatewayPID)

	return nil
}

func (m *Module) Start(ctx context.Context, c *runtime.Container) error { return nil }

func (m *Module) Stop(ctx context.Context, c *runtime.Container) error {
	rootCtx, err := runtime.Resolve[*actor.RootContext](c, runtime.ServiceActorRoot)
	if err == nil {
		if pid, e := runtime.Resolve[*actor.PID](c, runtime.ServiceGatewayPID); e == nil {
			rootCtx.Stop(pid)
		}
		if pid, e := runtime.Resolve[*actor.PID](c, runtime.ServiceSchedulerPID); e == nil {
			rootCtx.Stop(pid)
		}
		if pid, e := runtime.Resolve[*actor.PID](c, runtime.ServicePersistPID); e == nil {
			rootCtx.Stop(pid)
		}
	}

	if sys, err := runtime.Resolve[*actor.ActorSystem](c, runtime.ServiceActorSystem); err == nil {
		sys.Shutdown()
	}

	return nil
}
