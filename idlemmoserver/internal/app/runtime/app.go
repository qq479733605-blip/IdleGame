package runtime

import (
	"context"
	"fmt"
	"log"
)

// Module encapsulates a vertical slice of functionality. Modules configure their
// dependencies, start background workers, and can be stopped in reverse order when the
// application shuts down.
type Module interface {
	Name() string
	Configure(ctx context.Context, c *Container) error
	Start(ctx context.Context, c *Container) error
	Stop(ctx context.Context, c *Container) error
}

// App wires registered modules together and manages their lifecycle.
type App struct {
	container *Container
	modules   []Module
	started   []Module
}

// NewApp constructs an application runtime with an empty dependency container.
func NewApp() *App {
	return &App{container: NewContainer()}
}

// Container exposes the internal dependency container so tests can introspect state.
func (a *App) Container() *Container {
	return a.container
}

// Register appends a module to the startup sequence. Modules are configured and started
// in the order they are registered.
func (a *App) Register(module Module) {
	a.modules = append(a.modules, module)
}

// Run boots each module, waits for the context cancellation signal, then stops modules
// in reverse order. If configuration, startup, or shutdown fails the error is returned.
func (a *App) Run(ctx context.Context) error {
	if len(a.modules) == 0 {
		return fmt.Errorf("runtime: no modules registered")
	}

	for _, module := range a.modules {
		if err := module.Configure(ctx, a.container); err != nil {
			return fmt.Errorf("configure module %s: %w", module.Name(), err)
		}
	}

	a.started = a.started[:0]
	for _, module := range a.modules {
		if err := module.Start(ctx, a.container); err != nil {
			stopErr := a.stopReverse(ctx, len(a.started)-1)
			if stopErr != nil {
				return fmt.Errorf("start module %s: %w (cleanup error: %v)", module.Name(), err, stopErr)
			}
			return fmt.Errorf("start module %s: %w", module.Name(), err)
		}
		a.started = append(a.started, module)
	}

	<-ctx.Done()
	stopErr := a.stopReverse(ctx, len(a.started)-1)
	if stopErr != nil {
		return stopErr
	}

	return ctx.Err()
}

func (a *App) stopReverse(ctx context.Context, upto int) error {
	if upto < 0 {
		return nil
	}

	var firstErr error
	for i := upto; i >= 0; i-- {
		module := a.started[i]
		if err := module.Stop(ctx, a.container); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("stop module %s: %w", module.Name(), err)
			} else {
				log.Printf("runtime: stop error in module %s: %v", module.Name(), err)
			}
		}
	}
	a.started = a.started[:0]
	return firstErr
}
