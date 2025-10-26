package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"idlemmoserver/common"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

type Manager struct {
	nc       *nats.Conn
	loginURL string

	mu          sync.RWMutex
	connections map[string]*websocket.Conn
}

func NewManager(nc *nats.Conn, loginURL string) *Manager {
	return &Manager{
		nc:          nc,
		loginURL:    loginURL,
		connections: make(map[string]*websocket.Conn),
	}
}

func (m *Manager) Start(ctx context.Context) error {
	_, err := m.nc.Subscribe(common.SubjectGameToGate, func(msg *nats.Msg) {
		var payload common.GameToGateMessage
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			return
		}
		m.mu.RLock()
		conn := m.connections[payload.PlayerID]
		m.mu.RUnlock()
		if conn == nil {
			return
		}
		if err := conn.WriteJSON(payload.Message); err != nil {
			conn.Close()
			m.mu.Lock()
			delete(m.connections, payload.PlayerID)
			m.mu.Unlock()
		}
	})
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		m.nc.Drain()
	}()

	return nil
}

func (m *Manager) Register(playerID string, conn *websocket.Conn) {
	m.mu.Lock()
	m.connections[playerID] = conn
	m.mu.Unlock()
}

func (m *Manager) Unregister(playerID string) {
	m.mu.Lock()
	if conn, ok := m.connections[playerID]; ok {
		conn.Close()
		delete(m.connections, playerID)
	}
	m.mu.Unlock()
}

func (m *Manager) HandleConnection(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()

	playerID, err := m.authenticate(conn)
	if err != nil {
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	if err := m.ensurePlayer(ctx, playerID); err != nil {
		conn.WriteJSON(map[string]string{"error": err.Error()})
		return
	}

	m.Register(playerID, conn)
	defer m.Unregister(playerID)

	conn.WriteJSON(map[string]string{"type": "S_LoginOK", "player_id": playerID})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		envelope := common.GateClientMessage{PlayerID: playerID, Message: data}
		if err := m.nc.Publish(common.SubjectGateClient, common.MustMarshal(envelope)); err != nil {
			return
		}
	}
}

func (m *Manager) authenticate(conn *websocket.Conn) (string, error) {
	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", err
	}
	var msg common.ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	if msg.Type != "C_Login" {
		return "", errors.New("first message must be C_Login")
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		return "", err
	}
	if payload.Token == "" {
		return "", errors.New("missing token")
	}

	verifyReq := common.VerifyRequest{Token: payload.Token}
	body := common.MustMarshal(verifyReq)
	resp, err := http.Post(m.loginURL+"/verify", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("verify failed: %s", resp.Status)
	}
	var verifyResp common.VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return "", err
	}
	if !verifyResp.Valid {
		return "", errors.New("token invalid")
	}
	return verifyResp.PlayerID, nil
}

func (m *Manager) ensurePlayer(ctx context.Context, playerID string) error {
	req := common.GatewayEnsurePlayerRequest{PlayerID: playerID}
	msg, err := m.nc.RequestWithContext(ctx, common.SubjectGateEnsurePlayer, common.MustMarshal(req))
	if err != nil {
		return err
	}
	var resp common.GatewayEnsurePlayerResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}
	if !resp.Success {
		if resp.Error == "" {
			resp.Error = "unknown error"
		}
		return errors.New(resp.Error)
	}
	return nil
}
