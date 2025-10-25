package usermodule

import (
	"context"

	"idlemmoserver/internal/app/runtime"
	coreuser "idlemmoserver/internal/core/user"
	"idlemmoserver/internal/gateway"
	"idlemmoserver/internal/persist/userjson"
)

// Module provisions the user registration service and HTTP handler.
type Module struct {
	storagePath string
}

// New creates a user module backed by a JSON repository located at storagePath.
func New(storagePath string) *Module {
	return &Module{storagePath: storagePath}
}

func (m *Module) Name() string { return "user" }

func (m *Module) Configure(ctx context.Context, c *runtime.Container) error {
	repo := userjson.New(m.storagePath)
	service := coreuser.NewRegistrationService(repo)
	handler := gateway.NewUserHandler(service)

	c.MustProvide(runtime.ServiceUserRepo, repo)
	c.MustProvide(runtime.ServiceUserService, service)
	c.MustProvide(runtime.ServiceUserHandler, handler)
	return nil
}

func (m *Module) Start(ctx context.Context, c *runtime.Container) error { return nil }

func (m *Module) Stop(ctx context.Context, c *runtime.Container) error { return nil }
