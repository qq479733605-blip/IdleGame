package handler

import (
	"fmt"
	"log"

	"github.com/idle-server/common/nats"
)

// PersistHandler 持久化处理器基类
type PersistHandler struct {
	*BaseHandler
}

// NewPersistHandler 创建持久化处理器
func NewPersistHandler(name, messageType string, natsManager *nats.Manager) *PersistHandler {
	return &PersistHandler{
		BaseHandler: NewBaseHandler(name, messageType, natsManager),
	}
}

// SaveUserHandler 保存用户数据处理器
type SaveUserHandler struct {
	*PersistHandler
	saveFunc func(userID string, data interface{}) error
}

// NewSaveUserHandler 创建保存用户处理器
func NewSaveUserHandler(natsManager *nats.Manager, saveFunc func(string, interface{}) error) *SaveUserHandler {
	return &SaveUserHandler{
		PersistHandler: NewPersistHandler("SaveUserHandler", "C_SaveUser", natsManager),
		saveFunc:       saveFunc,
	}
}

// Handle 处理保存用户数据
func (h *SaveUserHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	userID, ok := reqData["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id")
	}

	data := reqData["data"]
	if data == nil {
		return nil, fmt.Errorf("missing data")
	}

	log.Printf("Processing save user data request for: %s", userID)

	if err := h.saveFunc(userID, data); err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"user_id": userID,
		"status":  "saved",
	}), nil
}

// LoadUserHandler 加载用户数据处理器
type LoadUserHandler struct {
	*PersistHandler
	loadFunc func(userID string) (interface{}, error)
}

// NewLoadUserHandler 创建加载用户处理器
func NewLoadUserHandler(natsManager *nats.Manager, loadFunc func(string) (interface{}, error)) *LoadUserHandler {
	return &LoadUserHandler{
		PersistHandler: NewPersistHandler("LoadUserHandler", "C_LoadUser", natsManager),
		loadFunc:       loadFunc,
	}
}

// Handle 处理加载用户数据
func (h *LoadUserHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	userID, ok := reqData["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id")
	}

	log.Printf("Processing load user data request for: %s", userID)

	data, err := h.loadFunc(userID)
	if err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"user_id": userID,
		"data":    data,
	}), nil
}

// SavePlayerHandler 保存玩家数据处理器
type SavePlayerHandler struct {
	*PersistHandler
	saveFunc func(playerID string, data interface{}) error
}

// NewSavePlayerHandler 创建保存玩家处理器
func NewSavePlayerHandler(natsManager *nats.Manager, saveFunc func(string, interface{}) error) *SavePlayerHandler {
	return &SavePlayerHandler{
		PersistHandler: NewPersistHandler("SavePlayerHandler", "C_SavePlayer", natsManager),
		saveFunc:       saveFunc,
	}
}

// Handle 处理保存玩家数据
func (h *SavePlayerHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	playerID, ok := reqData["player_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing player_id")
	}

	data := reqData["data"]
	if data == nil {
		return nil, fmt.Errorf("missing data")
	}

	log.Printf("Processing save player data request for: %s", playerID)

	if err := h.saveFunc(playerID, data); err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"player_id": playerID,
		"status":    "saved",
	}), nil
}

// LoadPlayerHandler 加载玩家数据处理器
type LoadPlayerHandler struct {
	*PersistHandler
	loadFunc func(playerID string) (interface{}, error)
}

// NewLoadPlayerHandler 创建加载玩家处理器
func NewLoadPlayerHandler(natsManager *nats.Manager, loadFunc func(string) (interface{}, error)) *LoadPlayerHandler {
	return &LoadPlayerHandler{
		PersistHandler: NewPersistHandler("LoadPlayerHandler", "C_LoadPlayer", natsManager),
		loadFunc:       loadFunc,
	}
}

// Handle 处理加载玩家数据
func (h *LoadPlayerHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	playerID, ok := reqData["player_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing player_id")
	}

	log.Printf("Processing load player data request for: %s", playerID)

	data, err := h.loadFunc(playerID)
	if err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"player_id": playerID,
		"data":      data,
	}), nil
}

// DeleteUserHandler 删除用户数据处理器
type DeleteUserHandler struct {
	*PersistHandler
	deleteFunc func(userID string) error
}

// NewDeleteUserHandler 创建删除用户处理器
func NewDeleteUserHandler(natsManager *nats.Manager, deleteFunc func(string) error) *DeleteUserHandler {
	return &DeleteUserHandler{
		PersistHandler: NewPersistHandler("DeleteUserHandler", "C_DeleteUser", natsManager),
		deleteFunc:     deleteFunc,
	}
}

// Handle 处理删除用户数据
func (h *DeleteUserHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	reqData, ok := request.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid request format")
	}

	userID, ok := reqData["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing user_id")
	}

	log.Printf("Processing delete user data request for: %s", userID)

	if err := h.deleteFunc(userID); err != nil {
		return ErrorResponseWithID(ctx.RequestID, err), nil
	}

	return SuccessResponseWithID(ctx.RequestID, map[string]interface{}{
		"user_id": userID,
		"status":  "deleted",
	}), nil
}
