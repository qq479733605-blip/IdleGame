package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/idle-server/common/nats"
	natsio "github.com/nats-io/nats.go"
)

// MessageContext 消息上下文
type MessageContext struct {
	MessageType string
	RequestID   string
	UserID      string
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// Response 响应结构
type Response struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Handler 消息处理器接口
type Handler interface {
	Handle(ctx *MessageContext, request interface{}) (*Response, error)
	GetMessageType() string
}

// BaseHandler 基础消息处理器
type BaseHandler struct {
	name        string
	messageType string
	natsManager *nats.Manager
	logger      *log.Logger
}

// NewBaseHandler 创建基础处理器
func NewBaseHandler(name, messageType string, natsManager *nats.Manager) *BaseHandler {
	return &BaseHandler{
		name:        name,
		messageType: messageType,
		natsManager: natsManager,
		logger:      log.Default(),
	}
}

// GetMessageType 获取消息类型
func (h *BaseHandler) GetMessageType() string {
	return h.messageType
}

// Handle 处理消息 - 基类实现
func (h *BaseHandler) Handle(ctx *MessageContext, request interface{}) (*Response, error) {
	h.logger.Printf("[%s] Handling message type: %s", h.name, h.messageType)
	return &Response{
		Success:   true,
		Data:      fmt.Sprintf("Processed by %s", h.name),
		RequestID: ctx.RequestID,
		Timestamp: time.Now().Unix(),
	}, nil
}

// HandlerRegistry 处理器注册表
type HandlerRegistry struct {
	handlers       map[string]Handler
	defaultHandler Handler
}

// NewHandlerRegistry 创建处理器注册表
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]Handler),
	}
}

// RegisterHandler 注册处理器
func (r *HandlerRegistry) RegisterHandler(handler Handler) {
	r.handlers[handler.GetMessageType()] = handler
	log.Printf("Registered handler for message type: %s", handler.GetMessageType())
}

// SetDefaultHandler 设置默认处理器
func (r *HandlerRegistry) SetDefaultHandler(handler Handler) {
	r.defaultHandler = handler
}

// GetHandler 获取处理器
func (r *HandlerRegistry) GetHandler(messageType string) (Handler, bool) {
	handler, exists := r.handlers[messageType]
	if !exists && r.defaultHandler != nil {
		return r.defaultHandler, true
	}
	return handler, exists
}

// MessageProcessor 统一消息处理器
type MessageProcessor struct {
	registry     *HandlerRegistry
	natsManager  *nats.Manager
	errorHandler ErrorHandler
}

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(natsManager *nats.Manager) *MessageProcessor {
	return &MessageProcessor{
		registry:     NewHandlerRegistry(),
		natsManager:  natsManager,
		errorHandler: NewDefaultErrorHandler(natsManager),
	}
}

// RegisterHandler 注册处理器
func (p *MessageProcessor) RegisterHandler(handler Handler) {
	p.registry.RegisterHandler(handler)
}

// SetDefaultHandler 设置默认处理器
func (p *MessageProcessor) SetDefaultHandler(handler Handler) {
	p.registry.SetDefaultHandler(handler)
}

// ProcessMessage 处理消息
func (p *MessageProcessor) ProcessMessage(msg *natsio.Msg) error {
	// 解析消息
	var request map[string]interface{}
	if err := json.Unmarshal(msg.Data, &request); err != nil {
		return p.errorHandler.HandleError(fmt.Errorf("invalid message format: %w", err), msg.Reply)
	}

	// 调试：记录收到的消息
	log.Printf("MessageProcessor: Received message: %+v", request)
	log.Printf("MessageProcessor: Raw message data: %s", string(msg.Data))

	// 提取消息类型
	messageType, ok := request["type"].(string)
	if !ok {
		log.Printf("MessageProcessor: Missing message type. Available keys: %v", getMapKeys(request))
		return p.errorHandler.HandleError(fmt.Errorf("missing message type"), msg.Reply)
	}

	// 创建消息上下文
	ctx := &MessageContext{
		MessageType: messageType,
		RequestID:   p.extractRequestID(request),
		UserID:      p.extractUserID(request),
		Timestamp:   time.Now(),
		Metadata:    p.extractMetadata(request),
	}

	// 获取处理器
	handler, exists := p.registry.GetHandler(messageType)
	if !exists {
		return p.errorHandler.HandleError(fmt.Errorf("no handler for message type: %s", messageType), msg.Reply)
	}

	// 处理消息
	response, err := handler.Handle(ctx, request)
	if err != nil {
		return p.errorHandler.HandleError(err, msg.Reply)
	}

	// 发送响应
	if msg.Reply != "" {
		if err := p.natsManager.PublishReply(msg.Reply, response); err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
	}

	return nil
}

// 辅助方法
func (p *MessageProcessor) extractRequestID(request map[string]interface{}) string {
	if reqID, ok := request["request_id"].(string); ok {
		return reqID
	}
	return ""
}

func (p *MessageProcessor) extractUserID(request map[string]interface{}) string {
	if userID, ok := request["user_id"].(string); ok {
		return userID
	}
	if playerID, ok := request["player_id"].(string); ok {
		return playerID
	}
	return ""
}

func (p *MessageProcessor) extractMetadata(request map[string]interface{}) map[string]interface{} {
	if metadata, ok := request["metadata"].(map[string]interface{}); ok {
		return metadata
	}
	return make(map[string]interface{})
}

// getMapKeys 获取map的所有键
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(err error, replySubject string) error
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	natsManager *nats.Manager
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler(natsManager *nats.Manager) *DefaultErrorHandler {
	return &DefaultErrorHandler{natsManager: natsManager}
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(err error, replySubject string) error {
	log.Printf("Error processing message: %v", err)

	if replySubject != "" {
		response := &Response{
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().Unix(),
		}
		return h.natsManager.PublishReply(replySubject, response)
	}

	return err
}

// 便捷的响应创建函数
func SuccessResponse(data interface{}) *Response {
	return &Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

func ErrorResponse(err error) *Response {
	return &Response{
		Success:   false,
		Error:     err.Error(),
		Timestamp: time.Now().Unix(),
	}
}

func SuccessResponseWithID(requestID string, data interface{}) *Response {
	return &Response{
		Success:   true,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
}

func ErrorResponseWithID(requestID string, err error) *Response {
	return &Response{
		Success:   false,
		Error:     err.Error(),
		RequestID: requestID,
		Timestamp: time.Now().Unix(),
	}
}
