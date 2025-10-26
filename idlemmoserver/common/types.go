package common

import "encoding/json"

type PlayerSnapshot struct {
	PlayerID          string           `json:"player_id"`
	SeqLevels         map[string]int   `json:"seq_levels"`
	Inventory         map[string]int64 `json:"inventory"`
	Exp               int64            `json:"exp"`
	OfflineLimitHours int64            `json:"offline_limit_hours"`
}

type ClientEnvelope struct {
	PlayerID string          `json:"player_id"`
	Payload  json.RawMessage `json:"payload"`
}

type ServerEnvelope struct {
	PlayerID string      `json:"player_id"`
	Payload  interface{} `json:"payload"`
}

// Simple client message types understood by the demo implementation.
type ClientMessage struct {
	Type   string          `json:"type"`
	Target string          `json:"target,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Token    string `json:"token,omitempty"`
	PlayerID string `json:"player_id,omitempty"`
}

type VerifyRequest struct {
	Token string `json:"token"`
}

type VerifyResponse struct {
	Valid    bool   `json:"valid"`
	PlayerID string `json:"player_id,omitempty"`
}

// GatewayToGameRequest ensures a player actor exists inside the game service.
type GatewayEnsurePlayerRequest struct {
	PlayerID string `json:"player_id"`
}

type GatewayEnsurePlayerResponse struct {
	PlayerID string `json:"player_id"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

type GateClientMessage struct {
	PlayerID string          `json:"player_id"`
	Message  json.RawMessage `json:"message"`
}

type GameToGateMessage struct {
	PlayerID string      `json:"player_id"`
	Message  interface{} `json:"message"`
}

type PersistSaveRequest struct {
	Snapshot PlayerSnapshot `json:"snapshot"`
}

type PersistLoadRequest struct {
	PlayerID string `json:"player_id"`
}

type PersistLoadResponse struct {
	Snapshot PlayerSnapshot `json:"snapshot"`
	Found    bool           `json:"found"`
}
