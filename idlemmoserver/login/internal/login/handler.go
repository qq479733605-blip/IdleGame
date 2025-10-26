package login

import (
	"net/http"

	"idlemmoserver/common"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, svc *Service) {
	r.POST("/register", func(c *gin.Context) {
		var req common.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := svc.Register(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	})

	r.POST("/login", func(c *gin.Context) {
		var req common.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp, err := svc.Login(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		status := http.StatusOK
		if !resp.Success {
			status = http.StatusUnauthorized
		}
		c.JSON(status, resp)
	})

	r.POST("/verify", func(c *gin.Context) {
		var req common.VerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp := svc.Verify(req.Token)
		status := http.StatusOK
		if !resp.Valid {
			status = http.StatusUnauthorized
		}
		c.JSON(status, resp)
	})
}
