package gate

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	HandleMessage(playerID string, data []byte) error
}

// ClientConnection 客户端连接 - 纯 WebSocket 连接管理
type ClientConnection struct {
	conn           *websocket.Conn
	playerID       string
	messageHandler MessageHandler
	onClose        func(playerID string)
	done           chan struct{}
	writeMutex     sync.Mutex
}

// NewClientConnection 创建新的客户端连接
func NewClientConnection(conn *websocket.Conn, messageHandler MessageHandler, onClose func(string)) *ClientConnection {
	return &ClientConnection{
		conn:           conn,
		messageHandler: messageHandler,
		onClose:        onClose,
		done:           make(chan struct{}),
	}
}

// Start 启动连接处理
func (c *ClientConnection) Start() {
	log.Printf("ClientConnection.Start() called for %s", c.conn.RemoteAddr().String())

	// 启动读取协程
	go c.readPump()

	log.Printf("ClientConnection started for %s", c.conn.RemoteAddr().String())
}

// Close 关闭连接
func (c *ClientConnection) Close() {
	select {
	case <-c.done:
		// 已经关闭了
		return
	default:
		close(c.done)
		c.conn.Close()
		if c.onClose != nil {
			c.onClose(c.playerID)
		}
	}
}

// Send 发送消息 - 线程安全
func (c *ClientConnection) Send(data []byte) {
	select {
	case <-c.done:
		return
	default:
		c.writeMutex.Lock()
		defer c.writeMutex.Unlock()

		err := c.conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("Failed to send message to client: %v", err)
			c.Close()
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

// readPump 读取消息循环
func (c *ClientConnection) readPump() {
	defer c.Close()

	log.Printf("Starting readPump for connection from %s", c.conn.RemoteAddr().String())

	for {
		// 检查连接是否还活着
		if c.conn == nil {
			log.Printf("Connection is nil, stopping readPump")
			return
		}

		// 读取消息
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			} else {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		log.Printf("Read WebSocket message: %s", string(data))

		// 处理消息
		if err := c.handleMessage(data); err != nil {
			log.Printf("Failed to handle message: %v", err)
		}
	}
}

// handleMessage 处理收到的消息 - 委托给消息处理器
func (c *ClientConnection) handleMessage(data []byte) error {
	if c.messageHandler == nil {
		log.Printf("No message handler set, ignoring message")
		return nil
	}

	return c.messageHandler.HandleMessage(c.playerID, data)
}
