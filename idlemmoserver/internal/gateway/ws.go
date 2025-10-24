package gateway

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
	"idlemmoserver/internal/actors"
)

// AttachWS 将 WS 连接绑定到 GatewayActor 的消息循环
func AttachWS(root *actor.RootContext, gatewayPID *actor.PID, conn *websocket.Conn) {
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
}
