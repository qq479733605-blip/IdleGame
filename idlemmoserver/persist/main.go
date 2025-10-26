package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"idlemmoserver/common"
	"idlemmoserver/persist/internal/persist"

	"github.com/nats-io/nats.go"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	repo, err := persist.NewJSONRepository("data")
	if err != nil {
		log.Fatalf("init repo: %v", err)
	}

	natsURL := getEnv("NATS_URL", common.DefaultNATSURL)
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer nc.Close()

	svc := persist.NewService(nc, repo)
	if err := svc.Start(ctx); err != nil {
		log.Fatalf("start persist service: %v", err)
	}

	log.Println("persist service started")
	<-ctx.Done()
	log.Println("persist service stopped")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
