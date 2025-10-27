package handler

import (
	"fmt"
	"log"

	"github.com/idle-server/common"
	"github.com/idle-server/common/nats"
)

// AuthHandler 认证处理器基类
type AuthHandler struct {
	*BaseHandler
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(name, messageType string, natsManager *nats.Manager) *AuthHandler {
	return &AuthHandler{
		BaseHandler: NewBaseHandler(name, messageType, natsManager),
	}
}

// LoginHandler 登录处理器
type LoginHandler struct {
	*AuthHandler
	authFunc func(username, password string) (*common.MsgAuthenticateUserResult, error)
}

// NewLoginHandler 创建登录处理器
func NewLoginHandler(natsManager *nats.Manager, authFunc func(string, string) (*common.MsgAuthenticateUserResult, error)) *LoginHandler {
	return &LoginHandler{
		AuthHandler: NewAuthHandler("LoginHandler", "C_Login", natsManager),
		authFunc:    authFunc,
	}
}

// Handle 处理登录请求
func (h *LoginHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	// 解析登录请求
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	username, ok := reqData["username"].(string)
	if !ok {
		return nil, fmt.Errorf("missing username")
	}

	password, ok := reqData["password"].(string)
	if !ok {
		return nil, fmt.Errorf("missing password")
	}

	log.Printf("Processing login request for user: %s", username)

	// 调用认证函数
	result, err := h.authFunc(username, password)
	if err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	if !result.Success {
		return ErrorResponseWithID(ctx.RequestID, fmt.Errorf(result.Message)), nil
	}

	return SuccessResponseWithID(ctx.RequestID, result), nil
}

// RegisterHandler 注册处理器
type RegisterHandler struct {
	*AuthHandler
	registerFunc func(username, password string) (*common.MsgRegisterUserResult, error)
}

// NewRegisterHandler 创建注册处理器
func NewRegisterHandler(natsManager *nats.Manager, registerFunc func(string, string) (*common.MsgRegisterUserResult, error)) *RegisterHandler {
	return &RegisterHandler{
		AuthHandler:  NewAuthHandler("RegisterHandler", "C_Register", natsManager),
		registerFunc: registerFunc,
	}
}

// Handle 处理注册请求
func (h *RegisterHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	username, ok := reqData["username"].(string)
	if !ok {
		return nil, fmt.Errorf("missing username")
	}

	password, ok := reqData["password"].(string)
	if !ok {
		return nil, fmt.Errorf("missing password")
	}

	log.Printf("Processing registration request for user: %s", username)

	result, err := h.registerFunc(username, password)
	if err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	if !result.Success {
		return ErrorResponseWithID(ctx.RequestID, fmt.Errorf(result.Message)), nil
	}

	return SuccessResponseWithID(ctx.RequestID, result), nil
}
