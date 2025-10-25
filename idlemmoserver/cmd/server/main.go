package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	actorsmod "idlemmoserver/internal/app/modules/actors"
	configmod "idlemmoserver/internal/app/modules/config"
	httptransport "idlemmoserver/internal/app/modules/transport/http"
	usermodule "idlemmoserver/internal/app/modules/user"
	"idlemmoserver/internal/app/runtime"
	"idlemmoserver/internal/logx"
)

func main() {
	logx.Init()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app := runtime.NewApp()
	app.Register(configmod.New("internal/domain/config_full.json", "internal/domain/equipment_config.json"))
	app.Register(actorsmod.New("saves"))
	app.Register(usermodule.New("saves/users.json"))
	app.Register(httptransport.New(":8080"))

	if err := app.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
