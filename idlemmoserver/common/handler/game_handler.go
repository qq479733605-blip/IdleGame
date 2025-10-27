package handler

import (
	"fmt"
	"log"

	"github.com/idle-server/common/nats"
)

// GameHandler 游戏处理器基类
type GameHandler struct {
	*BaseHandler
}

// NewGameHandler 创建游戏处理器
func NewGameHandler(name, messageType string, natsManager *nats.Manager) *GameHandler {
	return &GameHandler{
		BaseHandler: NewBaseHandler(name, messageType, natsManager),
	}
}

// PlayerConnectHandler 玩家连接处理器
type PlayerConnectHandler struct {
	*GameHandler
	connectFunc func(playerID string) error
}

// NewPlayerConnectHandler 创建玩家连接处理器
func NewPlayerConnectHandler(natsManager *nats.Manager, connectFunc func(string) error) *PlayerConnectHandler {
	return &PlayerConnectHandler{
		GameHandler: NewGameHandler("PlayerConnectHandler", "C_PlayerConnect", natsManager),
		connectFunc: connectFunc,
	}
}

// Handle 处理玩家连接
func (h *PlayerConnectHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	playerID, ok := reqData["player_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing player_id")
	}

	log.Printf("Processing player connect request for: %s", playerID)

	if err := h.connectFunc(playerID); err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"player_id": playerID,
		"status":    "connected",
	}), nil
}

// PlayerDisconnectHandler 玩家断开连接处理器
type PlayerDisconnectHandler struct {
	*GameHandler
	disconnectFunc func(playerID string) error
}

// NewPlayerDisconnectHandler 创建玩家断开连接处理器
func NewPlayerDisconnectHandler(natsManager *nats.Manager, disconnectFunc func(string) error) *PlayerDisconnectHandler {
	return &PlayerDisconnectHandler{
		GameHandler:    NewGameHandler("PlayerDisconnectHandler", "C_PlayerDisconnect", natsManager),
		disconnectFunc: disconnectFunc,
	}
}

// Handle 处理玩家断开连接
func (h *PlayerDisconnectHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	playerID, ok := reqData["player_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing player_id")
	}

	log.Printf("Processing player disconnect request for: %s", playerID)

	if err := h.disconnectFunc(playerID); err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"player_id": playerID,
		"status":    "disconnected",
	}), nil
}

// GameStateHandler 游戏状态处理器
type GameStateHandler struct {
	*GameHandler
	getStateFunc func(playerID string) (interface{}, error)
}

// NewGameStateHandler 创建游戏状态处理器
func NewGameStateHandler(natsManager *nats.Manager, getStateFunc func(string) (interface{}, error)) *GameStateHandler {
	return &GameStateHandler{
		GameHandler:  NewGameHandler("GameStateHandler", "C_GetState", natsManager),
		getStateFunc: getStateFunc,
	}
}

// Handle 处理获取游戏状态
func (h *GameStateHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	playerID, ok := reqData["player_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing player_id")
	}

	log.Printf("Processing get state request for player: %s", playerID)

	state, err := h.getStateFunc(playerID)
	if err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"player_id": playerID,
		"state":     state,
	}), nil
}

// GameActionHandler 游戏动作处理器
type GameActionHandler struct {
	*GameHandler
	actionFunc func(playerID, action string, params map[string]interface{}) (interface{}, error)
}

// NewGameActionHandler 创建游戏动作处理器
func NewGameActionHandler(natsManager *nats.Manager, actionFunc func(string, string, map[string]interface{}) (interface{}, error)) *GameActionHandler {
	return &GameActionHandler{
		GameHandler: NewGameHandler("GameActionHandler", "C_GameAction", natsManager),
		actionFunc:  actionFunc,
	}
}

// Handle 处理游戏动作
func (h *GameActionHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	playerID, ok := reqData["player_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing player_id")
	}

	action, ok := reqData["action"].(string)
	if !ok {
		return nil, fmt.Errorf("missing action")
	}

	params := make(map[string]interface{})
	if p, ok := reqData["params"].(map[string]interface{}); ok {
		params = p
	}

	log.Printf("Processing game action '%s' for player: %s", action, playerID)

	result, err := h.actionFunc(playerID, action, params)
	if err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"player_id": playerID,
		"action":    action,
		"result":    result,
	}), nil
}
