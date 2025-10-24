package main

import (
	"log"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"

	"idlemmoserver/internal/actors"
	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/gateway"
)

func main() {

	if err := domain.LoadConfig("internal/domain/config.json"); err != nil {
		log.Fatal(err)
	}
	// 1) ActorSystem
	sys := actor.NewActorSystem()
	root := sys.Root

	// 2) GatewayActor（WS 路由与玩家会话管理）
	gatewayPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return actors.NewGatewayActor(root)
	}))

	// 3) HTTP / WS 路由
	r := gin.Default()
	gateway.InitRoutes(r, root, gatewayPID)

	log.Println("server started :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
