package nats

import (
	"fmt"
	"log"

	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	HandleError(err error, replySubject string)
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	manager *Manager
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler(manager *Manager) *DefaultErrorHandler {
	return &DefaultErrorHandler{manager: manager}
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(err error, replySubject string) {
	log.Printf("Error processing message: %v", err)

	if replySubject != "" {
		errorMsg := map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		h.manager.PublishReply(replySubject, errorMsg)
	}
}

// MessageProcessor 消息处理器 - 处理特定类型的消息
type MessageProcessor struct {
	manager       *Manager
	errorHandler  ErrorHandler
	name          string
	unmarshalFunc func([]byte, interface{}) error
	processFunc   func(interface{}) (interface{}, error)
}

// NewMessageProcessor 创建消息处理器
func NewMessageProcessor(
	manager *Manager,
	name string,
	unmarshalFunc func([]byte, interface{}) error,
	processFunc func(interface{}) (interface{}, error),
) *MessageProcessor {
	return &MessageProcessor{
		manager:       manager,
		errorHandler:  NewDefaultErrorHandler(manager),
		name:          name,
		unmarshalFunc: unmarshalFunc,
		processFunc:   processFunc,
	}
}

// Handle 实现 MessageHandler 接口
func (p *MessageProcessor) Handle(msg *nats.Msg) error {
	log.Printf("[%s] Processing message on subject %s", p.name, msg.Subject)

	// 解析消息
	request := make(map[string]interface{})
	if err := p.unmarshalFunc(msg.Data, &request); err != nil {
		p.errorHandler.HandleError(fmt.Errorf("invalid message format: %w", err), msg.Reply)
		return err
	}

	// 处理消息
	response, err := p.processFunc(request)
	if err != nil {
		p.errorHandler.HandleError(err, msg.Reply)
		return err
	}

	// 发送响应
	if msg.Reply != "" {
		if err := p.manager.PublishReply(msg.Reply, response); err != nil {
			log.Printf("[%s] Failed to send reply: %v", p.name, err)
			return err
		}
	}

	log.Printf("[%s] Successfully processed message", p.name)
	return nil
}

// 便捷的处理器创建函数

// NewAuthMessageProcessor 创建认证消息处理器
func NewAuthMessageProcessor(manager *Manager, name string, processFunc func(interface{}) (interface{}, error)) *MessageProcessor {
	return NewMessageProcessor(
		manager,
		name,
		common.Unmarshal,
		processFunc,
	)
}

// NewGameMessageProcessor 创建游戏消息处理器
func NewGameMessageProcessor(manager *Manager, name string, processFunc func(interface{}) (interface{}, error)) *MessageProcessor {
	return NewMessageProcessor(
		manager,
		name,
		common.Unmarshal,
		processFunc,
	)
}

// NewPersistMessageProcessor 创建持久化消息处理器
func NewPersistMessageProcessor(manager *Manager, name string, processFunc func(interface{}) (interface{}, error)) *MessageProcessor {
	return NewMessageProcessor(
		manager,
		name,
		common.Unmarshal,
		processFunc,
	)
}

// BatchMessageProcessor 批量消息处理器
type BatchMessageProcessor struct {
	manager     *Manager
	processors  map[string]*MessageProcessor
	defaultProc *MessageProcessor
}

// NewBatchMessageProcessor 创建批量消息处理器
func NewBatchMessageProcessor(manager *Manager) *BatchMessageProcessor {
	return &BatchMessageProcessor{
		manager:    manager,
		processors: make(map[string]*MessageProcessor),
	}
}

// RegisterProcessor 注册处理器
func (b *BatchMessageProcessor) RegisterProcessor(subject string, processor *MessageProcessor) {
	b.processors[subject] = processor
}

// SetDefaultProcessor 设置默认处理器
func (b *BatchMessageProcessor) SetDefaultProcessor(processor *MessageProcessor) {
	b.defaultProc = processor
}

// Handle 实现 MessageHandler 接口
func (b *BatchMessageProcessor) Handle(msg *nats.Msg) error {
	// 尝试找到匹配的处理器
	if processor, exists := b.processors[msg.Subject]; exists {
		return processor.Handle(msg)
	}

	// 使用默认处理器
	if b.defaultProc != nil {
		return b.defaultProc.Handle(msg)
	}

	// 没有找到处理器
	err := fmt.Errorf("no processor registered for subject: %s", msg.Subject)
	log.Printf("Error: %v", err)
	return err
}
