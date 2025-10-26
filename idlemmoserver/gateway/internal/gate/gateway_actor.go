package gate

import (
	"encoding/json"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// GatewayActor 网关Actor
type GatewayActor struct {
	connections map[string]*ClientConnection
	nc          *nats.Conn
}

// NewGatewayActor 创建新的网关Actor
func NewGatewayActor(connections map[string]*ClientConnection) actor.Producer {
	return func() actor.Actor {
		return &GatewayActor{
			connections: connections,
		}
	}
}

// Receive 处理消息
func (a *GatewayActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgFromWS:
		a.handleFromWS(ctx, msg)
	case *common.MsgWSClosed:
		a.handleWSClosed(ctx, msg)
	case *common.MsgClientPayload:
		a.handleClientPayload(ctx, msg)
	case *common.MsgToClient:
		a.handleToClient(ctx, msg)
	default:
		log.Printf("GatewayActor: unknown message type %T", msg)
	}
}

// handleFromWS 处理来自WebSocket的消息
func (a *GatewayActor) handleFromWS(ctx actor.Context, msg *common.MsgFromWS) {
	// 解析客户端消息
	var clientMsg map[string]interface{}
	if err := json.Unmarshal(msg.Data, &clientMsg); err != nil {
		log.Printf("Failed to unmarshal client message: %v", err)
		return
	}

	// 获取消息类型
	msgType, ok := clientMsg["type"].(string)
	if !ok {
		log.Printf("Missing message type in client message")
		return
	}

	// 根据消息类型处理
	switch msgType {
	case "C_LoginAuth":
		a.handleLoginAuth(ctx, msg)
	case "C_Login":
		a.handleTokenLogin(ctx, msg)
	default:
		// 其他游戏消息，转发给游戏服务
		a.forwardToGameService(ctx, msg)
	}
}

// handleLoginAuth 处理登录认证
func (a *GatewayActor) handleLoginAuth(ctx actor.Context, msg *common.MsgFromWS) {
	var loginReq common.CLoginAuth
	if err := json.Unmarshal(msg.Data, &loginReq); err != nil {
		log.Printf("Failed to unmarshal login request: %v", err)
		return
	}

	// 发送NATS请求到登录服务
	authMsg := common.MsgAuthenticateUser{
		Username: loginReq.Username,
		Password: loginReq.Password,
	}

	// 发送NATS请求
	data, _ := json.Marshal(&authMsg)
	resp, err := a.nc.Request(common.LoginAuthSubject, data, 5*time.Second)
	if err != nil {
		log.Printf("Failed to send auth request: %v", err)
		return
	}

	var result common.MsgAuthenticateUserResult
	if err := common.Unmarshal(resp.Data, &result); err != nil {
		log.Printf("Failed to unmarshal auth response: %v", err)
		return
	}

	// 发送结果给客户端
	response := common.S_LoginOK{
		Type:     common.ServerMsgTypeLoginOK,
		Token:    result.Token,
		PlayerID: result.PlayerID,
	}

	data, _ = json.Marshal(response)
	msg.Conn.WriteMessage(websocket.TextMessage, data)
}

// handleTokenLogin 处理Token登录
func (a *GatewayActor) handleTokenLogin(ctx actor.Context, msg *common.MsgFromWS) {
	var loginReq common.CLogin
	if err := json.Unmarshal(msg.Data, &loginReq); err != nil {
		log.Printf("Failed to unmarshal token login request: %v", err)
		return
	}

	// TODO: 实现Token验证逻辑
	// 临时：直接成功
	response := common.S_LoginOK{
		Type:     common.ServerMsgTypeLoginOK,
		Token:    loginReq.Token,
		PlayerID: "temp_player_id", // 从Token解析
	}

	data, _ := json.Marshal(response)
	msg.Conn.WriteMessage(websocket.TextMessage, data)
}

// forwardToGameService 转发消息到游戏服务
func (a *GatewayActor) forwardToGameService(ctx actor.Context, msg *common.MsgFromWS) {
	// 创建客户端载荷消息
	payload := &common.MsgClientPayload{
		PlayerID: getPlayerIDFromConnection(msg.Conn), // 需要从连接中获取PlayerID
		Conn:     msg.Conn,
		Raw:      msg.Data,
	}

	ctx.Send(ctx.Self(), payload)
}

// handleWSClosed 处理WebSocket关闭
func (a *GatewayActor) handleWSClosed(ctx actor.Context, msg *common.MsgWSClosed) {
	// 从连接管理器中移除连接
	delete(a.connections, msg.Conn.RemoteAddr().String())

	// 通知游戏服务玩家离线
	playerID := getPlayerIDFromConnection(msg.Conn)
	if playerID != "" {
		offlineMsg := common.MsgPlayerOffline{}
		data, _ := json.Marshal(&offlineMsg)
		if err := a.nc.Publish(common.GamePlayerUnregisterSubject, data); err != nil {
			log.Printf("Failed to publish player offline message: %v", err)
		}
	}
}

// handleClientPayload 处理客户端载荷
func (a *GatewayActor) handleClientPayload(ctx actor.Context, msg *common.MsgClientPayload) {
	// 解析消息类型
	var clientMsg map[string]interface{}
	if err := json.Unmarshal(msg.Raw, &clientMsg); err != nil {
		log.Printf("Failed to unmarshal client payload: %v", err)
		return
	}

	msgType, ok := clientMsg["type"].(string)
	if !ok {
		return
	}

	// 根据消息类型转发到相应的NATS主题
	switch msgType {
	case "C_StartSeq":
		var startMsg common.CStartSeq
		json.Unmarshal(msg.Raw, &startMsg)
		gameMsg := common.MsgStartSequence{
			PlayerID: msg.PlayerID,
			SeqID:    startMsg.SeqID,
			Target:   startMsg.Target,
		}
		data, _ := json.Marshal(&gameMsg)
		a.nc.Publish(common.GameStartSequenceSubject, data)

	case "C_StopSeq":
		gameMsg := common.MsgStopSequence{
			PlayerID: msg.PlayerID,
		}
		data, _ := json.Marshal(&gameMsg)
		a.nc.Publish(common.GameStopSequenceSubject, data)
	}
}

// handleToClient 处理发送给客户端的消息
func (a *GatewayActor) handleToClient(ctx actor.Context, msg *common.MsgToClient) {
	// 查找对应的连接
	for _, conn := range a.connections {
		if conn.playerID == msg.PlayerID {
			conn.Send(msg.Data)
			break
		}
	}
}

// getPlayerIDFromConnection 从连接获取PlayerID（需要实现连接状态管理）
func getPlayerIDFromConnection(conn *websocket.Conn) string {
	// TODO: 实现从连接获取PlayerID的逻辑
	// 可以使用连接的额外属性或全局映射
	return ""
}
