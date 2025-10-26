package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"idlemmoserver/common"
	"idlemmoserver/gate/internal/gate"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	natsURL := getEnv("NATS_URL", common.DefaultNATSURL)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer nc.Close()

	mgr := gate.NewManager(nc, getEnv("LOGIN_URL", "http://127.0.0.1:8081"))
	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("start manager: %v", err)
	}

	r := gin.Default()
	r.Use(cors.Default())
	gate.RegisterRoutes(r, mgr)

	addr := getEnv("GATE_ADDR", ":8080")
	log.Printf("gateway listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
