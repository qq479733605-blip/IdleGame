package gate

import (
	"encoding/json"
	"time"

	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// NATSRequestHelper NATS请求助手
type NATSRequestHelper struct {
	nc *nats.Conn
}

// NewNATSRequestHelper 创建NATS请求助手
func NewNATSRequestHelper(nc *nats.Conn) *NATSRequestHelper {
	return &NATSRequestHelper{nc: nc}
}

// Request 发送NATS请求并等待响应
func (h *NATSRequestHelper) Request(subject string, msg interface{}, timeout time.Duration) (*nats.Msg, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// 发送请求
	return h.nc.Request(subject, data, timeout)
}

// Publish 发布NATS消息
func (h *NATSRequestHelper) Publish(subject string, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return h.nc.Publish(subject, data)
}

// Subscribe 订阅NATS主题
func (h *NATSRequestHelper) Subscribe(subject string, handler func(*nats.Msg)) (*nats.Subscription, error) {
	return h.nc.Subscribe(subject, handler)
}

// BroadcastToPlayer 向指定玩家广播消息
func (h *NATSRequestHelper) BroadcastToPlayer(playerID string, data []byte) error {
	msg := common.MsgToClient{
		PlayerID: playerID,
		Data:     data,
	}

	return h.Publish(common.GatewayBroadcastSubject, &msg)
}
