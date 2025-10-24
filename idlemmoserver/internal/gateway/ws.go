package gateway

import (
	"idlemmoserver/internal/actors"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

const (
	pingPeriod = 30 * time.Second // 客户端发心跳频率
	pongWait   = 60 * time.Second // 超时断开时间
)

// AttachWS 将 WS 连接绑定到 GatewayActor 的消息循环

func AttachWS(root *actor.RootContext, gatewayPID *actor.PID, conn *websocket.Conn) {
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// 读协程
	go func() {
		defer func() {
			conn.Close()
			root.Send(gatewayPID, &actors.MsgWSClosed{Conn: conn})
		}()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			root.Send(gatewayPID, &actors.MsgFromWS{Conn: conn, Data: data})
		}
	}()

	// 写协程（负责发 ping）
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		}
	}()
}
