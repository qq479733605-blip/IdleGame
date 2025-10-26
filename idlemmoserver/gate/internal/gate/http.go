package gate

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, mgr *Manager) {
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithCancel(c.Request.Context())
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
		mgr.HandleConnection(ctx, conn)
		cancel()
	})
}
