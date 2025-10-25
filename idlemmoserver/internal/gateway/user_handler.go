package gateway

import (
	"errors"
	"net/http"

	coreuser "idlemmoserver/internal/core/user"

	"github.com/gin-gonic/gin"
)

// UserHandler exposes HTTP endpoints for user-related operations.
type UserHandler struct {
	registration *coreuser.RegistrationService
}

// NewUserHandler builds a handler with the provided registration service.
func NewUserHandler(registration *coreuser.RegistrationService) *UserHandler {
	return &UserHandler{registration: registration}
}

// Register handles user sign-up requests.
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	created, err := h.registration.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, coreuser.ErrUserExists):
			c.JSON(http.StatusConflict, gin.H{"error": "username already registered"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"username":   created.Username,
		"created_at": created.CreatedAt,
	})
}
