package configmod

import (
	"context"

	"idlemmoserver/internal/app/runtime"
	"idlemmoserver/internal/domain"
)

// Module loads static domain configuration during bootstrap.
type Module struct {
	sequencesPath string
	equipmentPath string
}

// New creates a configuration module that loads the provided JSON assets.
func New(sequencesPath, equipmentPath string) *Module {
	return &Module{sequencesPath: sequencesPath, equipmentPath: equipmentPath}
}

func (m *Module) Name() string { return "config" }

func (m *Module) Configure(ctx context.Context, c *runtime.Container) error {
	if err := domain.LoadConfig(m.sequencesPath); err != nil {
		return err
	}
	if err := domain.LoadEquipmentConfig(m.equipmentPath); err != nil {
		return err
	}
	return nil
}

func (m *Module) Start(ctx context.Context, c *runtime.Container) error { return nil }

func (m *Module) Stop(ctx context.Context, c *runtime.Container) error { return nil }
