package gateway

import (
	"strings"
	"sync"
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/player"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

type ConnectionHandler struct {
	conn      *websocket.Conn
	playerPID *actor.PID
	root      *actor.RootContext
	playerID  string

	msgQueue  chan []byte
	readDone  chan struct{}
	writeDone chan struct{}
	stopChan  chan struct{}
}

var (
	playerActors   = make(map[string]*actor.PID)
	playerActorMux sync.RWMutex
	globalGateway  *actor.PID
)

func RegisterGateway(pid *actor.PID) {
	globalGateway = pid
}

func removePlayerActor(playerID string) {
	playerActorMux.Lock()
	delete(playerActors, playerID)
	playerActorMux.Unlock()
}

func getOrCreatePlayer(root *actor.RootContext, playerID string) *actor.PID {
	playerActorMux.RLock()
	if pid, ok := playerActors[playerID]; ok {
		playerActorMux.RUnlock()
		return pid
	}
	playerActorMux.RUnlock()
	if globalGateway == nil {
		return nil
	}
	resp, err := root.RequestFuture(globalGateway, &common.MsgEnsurePlayer{PlayerID: playerID}, 5*time.Second).Result()
	if err != nil {
		logx.Error("ensure player failed", "player", playerID, "err", err)
		return nil
	}
	ready, _ := resp.(*common.MsgPlayerReady)
	if ready == nil {
		return nil
	}
	playerActorMux.Lock()
	playerActors[playerID] = ready.PlayerPID
	playerActorMux.Unlock()
	return ready.PlayerPID
}

func NewConnectionHandler(root *actor.RootContext, conn *websocket.Conn, token string) (*ConnectionHandler, error) {
	playerID := parseToken(token)
	if playerID == "" {
		return nil, nil
	}
	handler := &ConnectionHandler{
		conn:      conn,
		root:      root,
		playerID:  playerID,
		msgQueue:  make(chan []byte, 256),
		readDone:  make(chan struct{}),
		writeDone: make(chan struct{}),
		stopChan:  make(chan struct{}),
	}
	pid := getOrCreatePlayer(root, playerID)
	if pid == nil {
		return nil, nil
	}
	handler.playerPID = pid
	root.Send(pid, &player.MsgAttachConn{Conn: conn, RequestState: true})
	return handler, nil
}

func parseToken(token string) string {
	if strings.HasPrefix(token, "mock-jwt-") {
		return strings.TrimPrefix(token, "mock-jwt-")
	}
	return token
}

func (h *ConnectionHandler) Start() {
	go h.readLoop()
	go h.writeLoop()
}

func (h *ConnectionHandler) Stop() {
	close(h.stopChan)
	if h.conn != nil {
		_ = h.conn.Close()
	}
	<-h.readDone
	<-h.writeDone
	if h.playerPID != nil {
		h.root.Send(h.playerPID, &player.MsgDetachConn{})
	}
}

func (h *ConnectionHandler) readLoop() {
	defer close(h.readDone)
	defer func() {
		if h.playerPID != nil {
			h.root.Send(h.playerPID, &common.MsgConnClosed{Conn: h.conn})
		}
	}()
	if h.conn == nil {
		return
	}
	h.conn.SetReadLimit(512)
	h.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	h.conn.SetPongHandler(func(string) error {
		if h.conn != nil {
			h.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
		return nil
	})
	for {
		select {
		case <-h.stopChan:
			return
		default:
		}
		_, data, err := h.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logx.Error("ws read", "err", err)
			}
			return
		}
		h.root.Send(h.playerPID, &common.MsgClientPayload{Conn: h.conn, Raw: data})
	}
}

func (h *ConnectionHandler) writeLoop() {
	defer close(h.writeDone)
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			if h.conn != nil {
				if err := h.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			}
		case data := <-h.msgQueue:
			if h.conn != nil {
				if err := h.conn.WriteMessage(websocket.TextMessage, data); err != nil {
					return
				}
			}
		}
	}
}

func (h *ConnectionHandler) SendToClient(data []byte) {
	select {
	case h.msgQueue <- data:
	case <-h.stopChan:
	default:
	}
}
