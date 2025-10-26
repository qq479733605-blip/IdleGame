package common

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/gorilla/websocket"
)

// ============ WebSocket 和客户端相关消息 ============

// MsgFromWS WebSocket原始数据 → GatewayActor
type MsgFromWS struct {
	Conn *websocket.Conn
	Data []byte
}

// MsgWSClosed WebSocket关闭
type MsgWSClosed struct{ Conn *websocket.Conn }

// MsgClientPayload Gateway → Player：客户端载荷
type MsgClientPayload struct {
	PlayerID string
	Conn     *websocket.Conn
	Raw      []byte
}

// MsgConnClosed 连接断开通知
type MsgConnClosed struct{ Conn *websocket.Conn }

// MsgToClient 发送给客户端的消息
type MsgToClient struct {
	PlayerID string
	Data     []byte
}

// ============ 玩家状态相关消息 ============

// MsgPlayerOffline 玩家离线
type MsgPlayerOffline struct{}

// MsgPlayerReconnect 玩家重连
type MsgPlayerReconnect struct{ Conn *websocket.Conn }

// MsgCheckExpire 检查过期
type MsgCheckExpire struct{}

// ============ 游戏逻辑相关消息 ============

// MsgStartSequence 开始修炼序列
type MsgStartSequence struct {
	PlayerID string
	SeqID    string
	Target   int64
}

// MsgStopSequence 停止修炼序列
type MsgStopSequence struct {
	PlayerID string
}

// MsgSequenceResult 修炼序列结果
type MsgSequenceResult struct {
	PlayerID   string
	SeqID      string
	Result     TickResult
	StopReason string // "stopped", "target_reached", "level_up"
}

// MsgSequenceTick 序列Tick
type MsgSequenceTick struct{}

// MsgSequenceStop 序列停止
type MsgSequenceStop struct{}

// ============ 持久化相关消息 ============

// MsgSavePlayer 保存玩家数据
type MsgSavePlayer struct {
	PlayerID   string
	PlayerData *PlayerData
}

// MsgLoadPlayer 加载玩家数据
type MsgLoadPlayer struct {
	PlayerID string
	ReplyTo  *actor.PID
}

// MsgLoadResult 加载结果
type MsgLoadResult struct {
	Data *PlayerData
	Err  error
}

// ============ 认证相关消息 ============

// MsgRegisterUser 注册用户
type MsgRegisterUser struct {
	Username string
	Password string
	ReplyTo  *actor.PID
}

// MsgRegisterUserResult 注册用户结果
type MsgRegisterUserResult struct {
	Success  bool
	Message  string
	PlayerID string
}

// MsgAuthenticateUser 认证用户
type MsgAuthenticateUser struct {
	Username string
	Password string
	ReplyTo  *actor.PID
}

// MsgAuthenticateUserResult 认证用户结果
type MsgAuthenticateUserResult struct {
	Success  bool
	Message  string
	PlayerID string
	Token    string
}

// MsgGetUserByPlayerID 根据PlayerID获取用户
type MsgGetUserByPlayerID struct {
	PlayerID string
	ReplyTo  *actor.PID
}

// MsgGetUserByPlayerIDResult 获取用户结果
type MsgGetUserByPlayerIDResult struct {
	User   *UserData
	Exists bool
}

// ============ Actor注册相关消息 ============

// MsgRegisterPlayer 注册玩家Actor
type MsgRegisterPlayer struct {
	PlayerID string
	PID      *actor.PID
}

// MsgUnregisterPlayer 注销玩家Actor
type MsgUnregisterPlayer struct{ PlayerID string }

// ============ 装备相关消息 ============

// MsgUpdateEquipmentBonus 更新装备加成
type MsgUpdateEquipmentBonus struct {
	PlayerID string
	Bonus    EquipmentBonus
}

// ============ 客户端消息类型 ============

// 客户端消息基类
type BaseClientMsg struct {
	Type string `json:"type"`
}

// CRegister 注册请求
type CRegister struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CLoginAuth 登录认证
type CLoginAuth struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CLogin Token登录
type CLogin struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

// CStartSeq 开始修炼
type CStartSeq struct {
	Type   string `json:"type"`
	SeqID  string `json:"seq_id"`
	Target int64  `json:"target"`
}

// CStopSeq 停止修炼
type CStopSeq struct {
	Type string `json:"type"`
}

// ============ 服务端消息类型 ============

// 服务端消息基类
type BaseServerMsg struct {
	Type string `json:"type"`
}

// S_RegisterOK 注册成功
type S_RegisterOK struct {
	Type string `json:"type"`
}

// S_LoginOK 登录成功
type S_LoginOK struct {
	Type     string `json:"type"`
	Token    string `json:"token"`
	PlayerID string `json:"player_id"`
}

// S_Error 错误消息
type S_Error struct {
	Type    string `json:"type"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// S_PlayerData 玩家数据
type S_PlayerData struct {
	Type     string      `json:"type"`
	PlayerID string      `json:"player_id"`
	Data     *PlayerData `json:"data"`
}

// S_SeqResult 修炼结果
type S_SeqResult struct {
	Type       string     `json:"type"`
	SeqID      string     `json:"seq_id"`
	Result     TickResult `json:"result"`
	StopReason string     `json:"stop_reason"`
}

// S_InventoryUpdate 库存更新
type S_InventoryUpdate struct {
	Type      string     `json:"type"`
	Inventory *Inventory `json:"inventory"`
}

// S_EquipmentUpdate 装备更新
type S_EquipmentUpdate struct {
	Type      string                    `json:"type"`
	Equipment map[string]EquipmentState `json:"equipment"`
}
