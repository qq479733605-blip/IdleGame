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

// MsgClientPayload Gateway → Game：客户端载荷
type MsgClientPayload struct {
	PlayerID string
	Data     []byte
}

// MsgConnClosed 连接断开通知
type MsgConnClosed struct{ Conn *websocket.Conn }

// MsgToClient 发送给客户端的消息
type MsgToClient struct {
	PlayerID string
	Data     []byte
}

// ============ Token验证相关消息 ============

// MsgVerifyToken 验证Token请求
type MsgVerifyToken struct {
	Token string
}

// MsgVerifyTokenResult 验证Token结果
type MsgVerifyTokenResult struct {
	Success  bool
	PlayerID string
	Error    string
}

// ============ 玩家状态相关消息 ============

// MsgPlayerOffline 玩家离线
type MsgPlayerOffline struct{}

// MsgPlayerReconnect 玩家重连
type MsgPlayerReconnect struct{ Conn *websocket.Conn }

// MsgCheckExpire 检查过期
type MsgCheckExpire struct{}

// ============ 玩家管理相关消息 ============

// MsgPlayerConnect 玩家连接
type MsgPlayerConnect struct {
	PlayerID string
	Conn     *websocket.Conn
}

// MsgPlayerDisconnect 玩家断开连接
type MsgPlayerDisconnect struct {
	PlayerID string
}

// ============ 持久化相关消息 ============

// MsgSavePlayer 保存玩家数据
type MsgSavePlayer struct {
	PlayerID   string
	PlayerData *PlayerData
}

// ============ 认证相关消息 ============

// MsgRegisterUser 注册用户
type MsgRegisterUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// MsgRegisterUserResult 注册用户结果
type MsgRegisterUserResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	PlayerID string `json:"playerId"`
}

// MsgAuthenticateUser 认证用户
type MsgAuthenticateUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// MsgAuthenticateUserResult 认证用户结果
type MsgAuthenticateUserResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	PlayerID string `json:"playerId"`
	Token    string `json:"token"`
}

// MsgSaveUser 保存用户数据
type MsgSaveUser struct {
	UserData *UserData
}

// MsgLoadUser 加载用户数据
type MsgLoadUser struct {
	Username string
	ReplyTo  *actor.PID
}

// MsgLoadUserResult 加载用户结果
type MsgLoadUserResult struct {
	UserData *UserData
	Err      error
}

// MsgUserExists 检查用户是否存在
type MsgUserExists struct {
	Username string
	ReplyTo  *actor.PID
}

// MsgUserExistsResult 用户存在结果
type MsgUserExistsResult struct {
	Exists bool
}

// ============ 持久化相关消息 ============

// MsgLoadPlayer 加载玩家数据
type MsgLoadPlayer struct {
	PlayerID string
}

// MsgLoadResult 加载结果
type MsgLoadResult struct {
	Data *PlayerData
	Err  error
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

// S_Pong 心跳响应
type S_Pong struct {
	Type string `json:"type"`
}

// S_PlayerData 玩家数据
type S_PlayerData struct {
	Type     string      `json:"type"`
	PlayerID string      `json:"player_id"`
	Data     *PlayerData `json:"data"`
}
