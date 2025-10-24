package gateway

import (
	"net/http"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// InitRoutes 注册 HTTP 与 WS 路由
func InitRoutes(r *gin.Engine, root *actor.RootContext, gatewayPID *actor.PID) {
	// 简单登录：返回一个 mock token
	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil || req.Username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"err": "bad request"})
			return
		}
		token := "mock-jwt-" + req.Username
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	// WebSocket：?token=xxx
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		AttachWS(root, gatewayPID, conn)
	})
}
