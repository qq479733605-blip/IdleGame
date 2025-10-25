package gateway

import (
	"idlemmoserver/internal/actors"
	"idlemmoserver/internal/logx"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

// ConnectionHandler 处理单个WebSocket连接，无锁设计
type ConnectionHandler struct {
	conn      *websocket.Conn
	playerPID *actor.PID
	root      *actor.RootContext
	playerID  string

	// 消息队列和生命周期管理（导出字段以便外部访问）
	MsgQueue  chan []byte
	ReadDone  chan struct{}
	WriteDone chan struct{}
	stopChan  chan struct{}
}

// NewConnectionHandler 创建新的连接处理器
// 全局PlayerActorMap 存储playerID -> PID的映射
var globalPlayerActorMap = make(map[string]*actor.PID)
var actorMapMutex sync.RWMutex

func GetOrCreatePlayerActor(root *actor.RootContext, playerID string) *actor.PID {
	actorMapMutex.RLock()
	if pid, exists := globalPlayerActorMap[playerID]; exists {
		actorMapMutex.RUnlock()
		logx.Info("PlayerActor已存在，复用", "playerID", playerID, "PID", pid)
		return pid
	}
	actorMapMutex.RUnlock()

	// 创建新的PlayerActor
	actorMapMutex.Lock()
	defer actorMapMutex.Unlock()

	// 双重检查，防止并发创建
	if pid, exists := globalPlayerActorMap[playerID]; exists {
		return pid
	}

	props := actor.PropsFromProducer(func() actor.Actor {
		return actors.NewPlayerActor(playerID, root, GlobalPersistPID, GlobalSchedulerPID)
	})
	pid := root.Spawn(props)
	globalPlayerActorMap[playerID] = pid

	logx.Info("创建新PlayerActor", "playerID", playerID, "PID", pid)
	return pid
}

func NewConnectionHandler(root *actor.RootContext, conn *websocket.Conn, token string) (*ConnectionHandler, error) {
	// 解析token获取playerID
	playerID := parseToken(token)
	if playerID == "" {
		return nil, nil
	}

	handler := &ConnectionHandler{
		conn:      conn,
		root:      root,
		playerID:  playerID,
		MsgQueue:  make(chan []byte, 256),
		ReadDone:  make(chan struct{}),
		WriteDone: make(chan struct{}),
		stopChan:  make(chan struct{}),
	}

	// 获取或创建PlayerActor（持久化）
	playerPID := GetOrCreatePlayerActor(root, playerID)
	handler.playerPID = playerPID

	// 通知PlayerActor有新连接，并请求当前状态
	root.Send(playerPID, &actors.MsgAttachConn{Conn: conn, RequestState: true})

	logx.Info("创建连接处理器", "playerID", playerID)
	return handler, nil
}

// Start 启动连接的读写循环
func (h *ConnectionHandler) Start() {
	// 启动读取循环
	go h.readLoop()
	// 启动写入循环
	go h.writeLoop()
}

// Stop 停止连接处理器
func (h *ConnectionHandler) Stop() {
	close(h.stopChan)

	// 安全关闭连接
	if h.conn != nil {
		h.conn.Close()
	}

	// 等待读写循环结束
	<-h.ReadDone
	<-h.WriteDone

	// 通知PlayerActor连接断开
	if h.playerPID != nil && h.root != nil {
		h.root.Send(h.playerPID, &actors.MsgDetachConn{})
	}
	logx.Info("连接处理器已停止", "playerID", h.playerID)
}

// readLoop 处理WebSocket消息读取，无锁
func (h *ConnectionHandler) readLoop() {
	defer close(h.ReadDone)

	// 检查连接是否为nil
	if h.conn == nil {
		logx.Warn("连接为nil，退出readLoop", "playerID", h.playerID)
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
				logx.Error("WebSocket读取错误", "err", err)
			}
			return
		}

		// 直接转发给PlayerActor，无需经过GatewayActor
		h.root.Send(h.playerPID, &actors.MsgClientPayload{
			Conn: h.conn,
			Raw:  data,
		})
	}
}

// writeLoop 处理WebSocket消息写入，无锁
func (h *ConnectionHandler) writeLoop() {
	defer close(h.WriteDone)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return

		case <-ticker.C:
			// 发送心跳ping
			if h.conn != nil {
				if err := h.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			} else {
				return
			}

		case data := <-h.MsgQueue:
			// 发送消息到客户端
			if h.conn != nil {
				if err := h.conn.WriteMessage(websocket.TextMessage, data); err != nil {
					return
				}
			} else {
				return
			}
		}
	}
}

// SendToClient 发送消息到客户端（线程安全）
func (h *ConnectionHandler) SendToClient(data []byte) {
	select {
	case h.MsgQueue <- data:
	case <-h.stopChan:
		// 连接已关闭
	default:
		// 队列满了，丢弃消息（可根据需求调整策略）
		logx.Warn("消息队列已满，丢弃消息", "playerID", h.playerID)
	}
}

// GetPlayerID 获取玩家ID
func (h *ConnectionHandler) GetPlayerID() string {
	return h.playerID
}

// ensurePlayerActor 确保PlayerActor存在
func ensurePlayerActor(root *actor.RootContext, playerID string) *actor.PID {
	// 这里需要创建或获取PlayerActor
	// 暂时使用简化的创建逻辑，实际中可能需要全局注册
	props := actor.PropsFromProducer(func() actor.Actor {
		// 使用全局persistPID
		return actors.NewPlayerActor(playerID, root, GlobalPersistPID, GlobalSchedulerPID)
	})
	return root.Spawn(props)
}

// 全局persistPID，在main.go中设置
var GlobalPersistPID *actor.PID
var GlobalSchedulerPID *actor.PID

// SetPersistPID 设置全局persistPID
func SetPersistPID(pid *actor.PID) {
	GlobalPersistPID = pid
}

// SetSchedulerPID 设置全局schedulerPID
func SetSchedulerPID(pid *actor.PID) {
	GlobalSchedulerPID = pid
}

// parseToken 简单token解析
func parseToken(token string) string {
	if len(token) > 9 && token[:9] == "mock-jwt-" {
		return token[9:]
	}
	return token
}
