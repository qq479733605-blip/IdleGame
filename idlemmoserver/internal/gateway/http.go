package gateway

import (
	"net/http"

	"idlemmoserver/internal/actors"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// ConnectionManager 管理所有活跃的连接
type ConnectionManager struct {
	connections map[*ConnectionHandler]bool
	addChan     chan *ConnectionHandler
	removeChan  chan *ConnectionHandler
	stopChan    chan struct{}
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	cm := &ConnectionManager{
		connections: make(map[*ConnectionHandler]bool),
		addChan:     make(chan *ConnectionHandler),
		removeChan:  make(chan *ConnectionHandler),
		stopChan:    make(chan struct{}),
	}
	go cm.run()
	return cm
}

// run 运行连接管理器
func (cm *ConnectionManager) run() {
	for {
		select {
		case conn := <-cm.addChan:
			cm.connections[conn] = true
		case conn := <-cm.removeChan:
			delete(cm.connections, conn)
		case <-cm.stopChan:
			// 关闭所有连接
			for conn := range cm.connections {
				conn.Stop()
			}
			return
		}
	}
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn *ConnectionHandler) {
	cm.addChan <- conn
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(conn *ConnectionHandler) {
	cm.removeChan <- conn
}

// Stop 停止连接管理器
func (cm *ConnectionManager) Stop() {
	close(cm.stopChan)
}

// 全局连接管理器
var globalConnManager *ConnectionManager

// InitRoutes 注册 HTTP 与 WS 路由
func InitRoutes(r *gin.Engine, root *actor.RootContext, authPID *actor.PID) {
	// 初始化连接管理器
	globalConnManager = NewConnectionManager()

	// 用户注册
	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
			return
		}
		if req.Username == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
			return
		}

		// 异步处理注册请求
		responseChan := make(chan *actors.MsgRegisterUserResult)
		root.Request(authPID, &actors.MsgRegisterUser{
			Username: req.Username,
			Password: req.Password,
			ReplyTo: root.Spawn(actor.PropsFromProducer(func() actor.Actor {
				return &responseActor{responseChan: responseChan}
			})),
		})

		// 等待响应
		select {
		case result := <-responseChan:
			if result.Success {
				c.JSON(http.StatusOK, gin.H{
					"success":   true,
					"message":   result.Message,
					"player_id": result.PlayerID,
				})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   result.Message,
				})
			}
		case <-c.Request.Context().Done():
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "请求超时"})
		}
	})

	// 用户登录
	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
			return
		}
		if req.Username == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
			return
		}

		// 异步处理登录请求
		responseChan := make(chan *actors.MsgAuthenticateUserResult)
		root.Request(authPID, &actors.MsgAuthenticateUser{
			Username: req.Username,
			Password: req.Password,
			ReplyTo: root.Spawn(actor.PropsFromProducer(func() actor.Actor {
				return &responseActor{responseChan: responseChan}
			})),
		})

		// 等待响应
		select {
		case result := <-responseChan:
			if result.Success {
				// 生成 token
				token := "mock-jwt-" + result.PlayerID
				c.JSON(http.StatusOK, gin.H{
					"success":   true,
					"message":   result.Message,
					"token":     token,
					"player_id": result.PlayerID,
				})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   result.Message,
				})
			}
		case <-c.Request.Context().Done():
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "请求超时"})
		}
	})

	// WebSocket：?token=xxx
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		token := c.Query("token")
		HandleWebSocketConnection(root, conn, token)
	})
}

// HandleWebSocketConnection 处理WebSocket连接（新架构）
func HandleWebSocketConnection(root *actor.RootContext, conn *websocket.Conn, token string) {
	// 创建连接处理器
	handler, err := NewConnectionHandler(root, conn, token)
	if err != nil {
		conn.Close()
		return
	}

	// 检查handler是否为nil（token无效的情况）
	if handler == nil {
		logx.Info("无效的token，关闭连接", "token", token)
		conn.Close()
		return
	}

	// 添加到全局连接管理器
	globalConnManager.AddConnection(handler)

	// 启动连接处理器
	handler.Start()

	// 设置连接关闭回调 - 等待连接自然断开
	go func() {
		// 等待readLoop结束（连接自然断开）
		<-handler.ReadDone
		// 连接断开后，从管理器中移除
		globalConnManager.RemoveConnection(handler)
	}()
}

// responseActor 用于处理异步响应的临时 Actor
type responseActor struct {
	responseChan interface{}
}

func (r *responseActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *actors.MsgRegisterUserResult:
		if ch, ok := r.responseChan.(chan *actors.MsgRegisterUserResult); ok {
			ch <- m
		}
	case *actors.MsgAuthenticateUserResult:
		if ch, ok := r.responseChan.(chan *actors.MsgAuthenticateUserResult); ok {
			ch <- m
		}
	}
}
