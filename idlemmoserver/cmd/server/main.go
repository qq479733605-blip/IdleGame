package main

import (
	"log"
	"time"

	"idlemmoserver/internal/gateway"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/persist"
	"idlemmoserver/internal/player"
	"idlemmoserver/internal/scheduler"
	"idlemmoserver/internal/sequence"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	logx.Init()

	if err := sequence.LoadConfig("internal/sequence/config_full.json"); err != nil {
		log.Fatal(err)
	}
	if err := player.LoadEquipmentConfig("internal/player/equipment_config.json"); err != nil {
		log.Fatal(err)
	}

	sys := actor.NewActorSystem()
	root := sys.Root

	repo := persist.NewJSONRepo("saves")
	persistPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return persist.NewActor(repo)
	}))

	userRepo := persist.NewJSONUserRepo("users")
	authPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return gateway.NewAuthActor(userRepo)
	}))

	schedulerPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return scheduler.NewActor(200 * time.Millisecond)
	}))

	services := &player.Services{PersistPID: persistPID, SchedulerPID: schedulerPID}

	gatewayPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return gateway.NewGatewayActor(root, services)
	}))
	services.GatewayPID = gatewayPID

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	gateway.InitRoutes(r, root, authPID, gatewayPID)

	log.Println("server started on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
