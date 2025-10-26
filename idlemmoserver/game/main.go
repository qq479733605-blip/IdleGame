package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"idlemmoserver/common"
	game "idlemmoserver/game/internal/game"

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

	server := game.NewServer(nc)
	if err := server.Start(ctx); err != nil {
		log.Fatalf("start game server: %v", err)
	}

	log.Println("game service started")
	<-ctx.Done()
	log.Println("game service stopped")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
