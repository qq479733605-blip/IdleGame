package common

// 常量定义

// 服务器端口
const (
	LoginServicePort   = 8001
	GatewayServicePort = 8002
	GameServicePort    = 8003
	PersistServicePort = 8004
)

// NATS配置
const (
	NATSURL     = "nats://localhost:4222"
	ClusterName = "idle-mmso-cluster"
)

// WebSocket配置
const (
	WSPingInterval   = 30 // 秒
	WSWriteWait      = 10 // 秒
	WSPongWait       = 60 // 秒
	WSMaxMessageSize = 512
)

// 游戏配置
const (
	DefaultOfflineLimitHours = 24 // 小时
	DefaultInventorySize     = 30
	DefaultTickInterval      = 1 // 秒
)

// 错误码
const (
	ErrorCodeSuccess       = 0
	ErrorCodeAuthFailed    = 1001
	ErrorCodeUserExists    = 1002
	ErrorCodeUserNotFound  = 1003
	ErrorCodeInvalidToken  = 1004
	ErrorCodeInvalidData   = 2001
	ErrorCodeInternalError = 5000
)

// 消息类型
const (
	// 客户端消息类型
	ClientMsgTypeRegister  = "C_Register"
	ClientMsgTypeLoginAuth = "C_LoginAuth"
	ClientMsgTypeLogin     = "C_Login"
	ClientMsgTypeStartSeq  = "C_StartSeq"
	ClientMsgTypeStopSeq   = "C_StopSeq"

	// 服务端消息类型
	ServerMsgTypeRegisterOK      = "S_RegisterOK"
	ServerMsgTypeLoginOK         = "S_LoginOK"
	ServerMsgTypeError           = "S_Error"
	ServerMsgTypePlayerData      = "S_PlayerData"
	ServerMsgTypeSeqResult       = "S_SeqResult"
	ServerMsgTypeInventoryUpdate = "S_InventoryUpdate"
	ServerMsgTypeEquipmentUpdate = "S_EquipmentUpdate"
)

// 服务名称
const (
	ServiceNameLogin   = "login"
	ServiceNameGateway = "gateway"
	ServiceNameGame    = "game"
	ServiceNamePersist = "persist"
)
