package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Manager NATS 管理器 - 统一所有 NATS 通信逻辑
type Manager struct {
	nc *nats.Conn
}

// NewManager 创建 NATS 管理器
func NewManager(url string) (*Manager, error) {
	nc, err := nats.Connect(url,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", url, err)
	}

	log.Printf("Successfully connected to NATS at %s", url)
	return &Manager{nc: nc}, nil
}

// Subscribe 订阅 NATS 主题
func (m *Manager) Subscribe(subject string, handler MessageHandler) (*nats.Subscription, error) {
	sub, err := m.nc.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler.Handle(msg); err != nil {
			log.Printf("Error handling message on subject %s: %v", subject, err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	log.Printf("Subscribed to NATS subject: %s", subject)
	return sub, nil
}

// Request 发送 NATS 请求并等待响应
func (m *Manager) Request(subject string, msg interface{}, timeout time.Duration) (*nats.Msg, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	response, err := m.nc.Request(subject, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("NATS request failed on subject %s: %w", subject, err)
	}

	return response, nil
}

// Publish 发布 NATS 消息
func (m *Manager) Publish(subject string, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := m.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to subject %s: %w", subject, err)
	}

	return nil
}

// PublishReply 发布回复消息
func (m *Manager) PublishReply(replySubject string, msg interface{}) error {
	if replySubject == "" {
		return fmt.Errorf("reply subject is empty")
	}
	return m.Publish(replySubject, msg)
}

// GetConnection 获取原生 NATS 连接（用于特殊操作）
func (m *Manager) GetConnection() *nats.Conn {
	return m.nc
}

// Status 获取连接状态
func (m *Manager) Status() nats.Status {
	return m.nc.Status()
}

// IsConnected 检查是否已连接
func (m *Manager) IsConnected() bool {
	return m.nc.IsConnected()
}

// Close 关闭 NATS 连接
func (m *Manager) Close() {
	if m.nc != nil {
		m.nc.Close()
		log.Printf("NATS connection closed")
	}
}

// MessageHandler NATS 消息处理器接口
type MessageHandler interface {
	Handle(msg *nats.Msg) error
}

// BaseMessageHandler 基础消息处理器
type BaseMessageHandler struct {
	name string
}

// NewBaseMessageHandler 创建基础消息处理器
func NewBaseMessageHandler(name string) *BaseMessageHandler {
	return &BaseMessageHandler{name: name}
}

// Handle 处理消息 - 基类实现
func (h *BaseMessageHandler) Handle(msg *nats.Msg) error {
	log.Printf("[%s] Received message on subject %s", h.name, msg.Subject)
	return nil
}

// RequestWithReply 使用通用消息结构发送请求并等待响应
func (m *Manager) RequestWithReply(subject string, request interface{}, response interface{}, timeout time.Duration) error {
	msg, err := m.Request(subject, request, timeout)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(msg.Data, response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// 预定义的常用服务主题订阅方法
func (m *Manager) SubscribeToAuth(service string, handler MessageHandler) error {
	subject := fmt.Sprintf("auth.%s.*", service)
	_, err := m.Subscribe(subject, handler)
	return err
}

func (m *Manager) SubscribeToGame(service string, handler MessageHandler) error {
	subject := fmt.Sprintf("game.%s.*", service)
	_, err := m.Subscribe(subject, handler)
	return err
}

func (m *Manager) SubscribeToPersist(service string, handler MessageHandler) error {
	subject := fmt.Sprintf("persist.%s.*", service)
	_, err := m.Subscribe(subject, handler)
	return err
}

func (m *Manager) SubscribeToGateway(service string, handler MessageHandler) error {
	subject := fmt.Sprintf("gateway.%s.*", service)
	_, err := m.Subscribe(subject, handler)
	return err
}
