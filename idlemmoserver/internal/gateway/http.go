package gateway

import (
	"net/http"
	"sync"

	"idlemmoserver/internal/common"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ConnectionManager struct {
	connections map[*ConnectionHandler]bool
	addChan     chan *ConnectionHandler
	removeChan  chan *ConnectionHandler
	stopChan    chan struct{}
	once        sync.Once
}

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

func (cm *ConnectionManager) run() {
	for {
		select {
		case conn := <-cm.addChan:
			cm.connections[conn] = true
			conn.Start()
		case conn := <-cm.removeChan:
			if cm.connections[conn] {
				conn.Stop()
				delete(cm.connections, conn)
			}
		case <-cm.stopChan:
			for conn := range cm.connections {
				conn.Stop()
			}
			return
		}
	}
}

func (cm *ConnectionManager) AddConnection(conn *ConnectionHandler) {
	cm.addChan <- conn
}

func (cm *ConnectionManager) RemoveConnection(conn *ConnectionHandler) {
	cm.removeChan <- conn
}

func (cm *ConnectionManager) Stop() {
	cm.once.Do(func() { close(cm.stopChan) })
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func InitRoutes(r *gin.Engine, root *actor.RootContext, authPID *actor.PID, gatewayPID *actor.PID) {
	manager := NewConnectionManager()
	RegisterGateway(gatewayPID)

	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		if req.Username == "" || req.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
			return
		}
		responseChan := make(chan *common.MsgRegisterUserResult, 1)
		pid := root.Spawn(actor.PropsFromProducer(func() actor.Actor { return &responseActor{responseChan: responseChan} }))
		root.Send(authPID, &common.MsgRegisterUser{Username: req.Username, Password: req.Password, ReplyTo: pid})
		result := <-responseChan
		if result.Success {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": result.Message, "player_id": result.PlayerID})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": result.Message})
		}
	})

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		responseChan := make(chan *common.MsgAuthenticateUserResult, 1)
		pid := root.Spawn(actor.PropsFromProducer(func() actor.Actor { return &responseActor{authChan: responseChan} }))
		root.Send(authPID, &common.MsgAuthenticateUser{Username: req.Username, Password: req.Password, ReplyTo: pid})
		result := <-responseChan
		if result.Success {
			token := "mock-jwt-" + result.PlayerID
			c.JSON(http.StatusOK, gin.H{"success": true, "message": result.Message, "token": token, "player_id": result.PlayerID})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": result.Message})
		}
	})

	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		token := c.Query("token")
		handler, err := NewConnectionHandler(root, conn, token)
		if err != nil || handler == nil {
			conn.Close()
			return
		}
		manager.AddConnection(handler)
	})
}

type responseActor struct {
	responseChan chan *common.MsgRegisterUserResult
	authChan     chan *common.MsgAuthenticateUserResult
}

func (a *responseActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgRegisterUserResult:
		a.responseChan <- msg
		ctx.Stop(ctx.Self())
	case *common.MsgAuthenticateUserResult:
		a.authChan <- msg
		ctx.Stop(ctx.Self())
	}
}
