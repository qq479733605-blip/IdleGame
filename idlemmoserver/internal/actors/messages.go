package actors

import (
	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/persist"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

// —— Gateway ↔︎ Player / 入口消息 ——

// WebSocket 原始数据 → GatewayActor
type MsgFromWS struct {
	Conn *websocket.Conn
	Data []byte
}

// WebSocket 关闭
type MsgWSClosed struct{ Conn *websocket.Conn }

// Gateway → Player：把原始 WS payload 透传给 PlayerActor（让它自己解析）
type MsgClientPayload struct {
	Conn *websocket.Conn
	Raw  []byte
}

// 连接断开通知给 PlayerActor（可与 MsgWSClosed 配合）
type MsgConnClosed struct{ Conn *websocket.Conn }

// —— 在线/离线/重连/过期检查 ——

// 连接断了：进入离线结算
type MsgPlayerOffline struct{}

// 新连接来了：重连绑定
type MsgPlayerReconnect struct{ Conn *websocket.Conn }

// 定期检查离线是否超过上限
type MsgCheckExpire struct{}

// —— Player ↔︎ Persist ——

// 在 persist_actor.go 里已经定义了：

// 消息
type MsgSavePlayer struct {
	PlayerID          string
	SeqLevels         map[string]int
	Inventory         *domain.Inventory
	Exp               int64
	Equipment         map[string]domain.EquipmentState
	OfflineLimitHours int64
}
type MsgLoadPlayer struct {
	PlayerID string
	ReplyTo  *actor.PID
}
type MsgLoadResult struct {
	Data *persist.PlayerData
	Err  error
}

type MsgRegisterPlayer struct {
	PlayerID string
	PID      *actor.PID
}
type MsgUnregisterPlayer struct{ PlayerID string }

// 这些保持不动即可。
type baseMsg struct {
	Type string `json:"type"`
}
type CLogin struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}
type CStartSeq struct {
	Type   string `json:"type"`
	SeqID  string `json:"seq_id"`
	Target int64  `json:"target"`
}
type CStopSeq struct {
	Type string `json:"type"`
}

type SeqTick struct{}
type SeqStop struct{}

type MsgUpdateEquipmentBonus struct {
	Bonus domain.EquipmentBonus
}

// —— 用户认证相关消息 ——
type MsgRegisterUser struct {
	Username string
	Password string
	ReplyTo  *actor.PID
}

type MsgRegisterUserResult struct {
	Success  bool
	Message  string
	PlayerID string
}

type MsgAuthenticateUser struct {
	Username string
	Password string
	ReplyTo  *actor.PID
}

type MsgAuthenticateUserResult struct {
	Success  bool
	Message  string
	PlayerID string
}

type MsgGetUserByPlayerID struct {
	PlayerID string
	ReplyTo  *actor.PID
}

type MsgGetUserByPlayerIDResult struct {
	User   *domain.UserData
	Exists bool
}

// 客户端消息类型
type CRegister struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CLoginAuth struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}
