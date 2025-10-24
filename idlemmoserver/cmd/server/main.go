package main

import (
	"idlemmoserver/internal/actors"
	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/gateway"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/persist"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {

	logx.Init()

	// 1) 加载表驱动配置
	if err := domain.LoadConfig("internal/domain/config.json"); err != nil {
		log.Fatal(err)
	}

	// 2) ActorSystem
	sys := actor.NewActorSystem()
	root := sys.Root

	// 3) 持久化：JSONRepo + PersistActor
	repo := persist.NewJSONRepo("saves")
	persistPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return actors.NewPersistActor(repo)
	}))

	// 4) GatewayActor（传入 persistPID，便于 PlayerActor 保存/加载）
	gatewayPID := root.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return actors.NewGatewayActor(root, persistPID)
	}))

	// 5) HTTP/WS 路由
	r := gin.Default()
	// ✅ 加上 CORS 中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	gateway.InitRoutes(r, root, gatewayPID)

	log.Println("✅ loaded sequences config, server started :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
