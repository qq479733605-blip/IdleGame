package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type UserData struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	PlayerID  string    `json:"player_id"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

func GeneratePlayerID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func HashPassword(password string) string {
	return fmt.Sprintf("%x", password)
}

func VerifyPassword(password, hash string) bool {
	return HashPassword(password) == hash
}
