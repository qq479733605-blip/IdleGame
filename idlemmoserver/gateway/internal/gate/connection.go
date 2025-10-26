package gate

import (
	"log"
	"net/http"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// ClientConnection 客户端连接
type ClientConnection struct {
	conn       *websocket.Conn
	playerID   string
	system     *actor.ActorSystem
	gatewayPID *actor.PID
	nc         *nats.Conn
	done       chan struct{}
}

// NewClientConnection 创建新的客户端连接
func NewClientConnection(conn *websocket.Conn, system *actor.ActorSystem, gatewayPID *actor.PID, nc *nats.Conn) *ClientConnection {
	return &ClientConnection{
		conn:       conn,
		system:     system,
		gatewayPID: gatewayPID,
		nc:         nc,
		done:       make(chan struct{}),
	}
}

// Start 启动连接处理
func (c *ClientConnection) Start() {
	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(common.WSPongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(common.WSPongWait))
		return nil
	})

	// 启动读取协程
	go c.readPump()

	// 启动心跳协程
	go c.writePump()
}

// Close 关闭连接
func (c *ClientConnection) Close() {
	close(c.done)
	c.conn.Close()
}

// Send 发送消息
func (c *ClientConnection) Send(data []byte) {
	select {
	case <-c.done:
		return
	default:
		err := c.conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Failed to send message to client: %v", err)
			c.Close()
		}
	}
}

// readPump 读取消息循环
func (c *ClientConnection) readPump() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		default:
			// 读取消息
			_, data, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// 发送消息给网关Actor
			msg := &common.MsgFromWS{
				Conn: c.conn,
				Data: data,
			}
			// 创建临时context来发送消息
			tempProps := actor.PropsFromProducer(func() actor.Actor {
				return &MessageForwarder{
					targetPID: c.gatewayPID,
					message:   msg,
				}
			})
			tempPID := c.system.Root.Spawn(tempProps)
			c.system.Root.Send(tempPID, struct{}{})
		}
	}
}

// writePump 写入消息循环（心跳）
func (c *ClientConnection) writePump() {
	ticker := time.NewTicker(common.WSPingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			// 发送心跳
			c.conn.SetWriteDeadline(time.Now().Add(common.WSWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SetPlayerID 设置玩家ID
func (c *ClientConnection) SetPlayerID(playerID string) {
	c.playerID = playerID
}

// GetPlayerID 获取玩家ID
func (c *ClientConnection) GetPlayerID() string {
	return c.playerID
}

// MessageForwarder 消息转发器
type MessageForwarder struct {
	targetPID *actor.PID
	message   interface{}
}

// Receive 处理消息
func (m *MessageForwarder) Receive(ctx actor.Context) {
	switch ctx.Message().(type) {
	case struct{}:
		// 启动信号，转发消息
		ctx.Send(m.targetPID, m.message)
		ctx.Stop(ctx.Self())
	}
}

// WebSocketUpgrader WebSocket升级器
type WebSocketUpgrader struct {
	upgrader websocket.Upgrader
}

// NewWebSocketUpgrader 创建WebSocket升级器
func NewWebSocketUpgrader() WebSocketUpgrader {
	return WebSocketUpgrader{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中应该检查Origin
				return true
			},
		},
	}
}

// Upgrade 升级HTTP连接到WebSocket
func (u *WebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return u.upgrader.Upgrade(w, r, nil)
}
