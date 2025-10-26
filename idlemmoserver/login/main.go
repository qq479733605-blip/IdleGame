package main

import (
	"log"
	"os"

	"idlemmoserver/login/internal/login"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	repo, err := login.NewJSONUserRepository("data")
	if err != nil {
		log.Fatalf("init user repo: %v", err)
	}
	svc := login.NewService(repo)

	r := gin.Default()
	r.Use(cors.Default())
	login.RegisterRoutes(r, svc)

	addr := getEnv("LOGIN_ADDR", ":8081")
	log.Printf("login service listening on %s", addr)
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
