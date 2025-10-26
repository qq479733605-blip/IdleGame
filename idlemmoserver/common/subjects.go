package common

// NATS主题定义
const (
	// ============ 登录服务相关 ============
	LoginAuthSubject     = "login.auth"
	LoginRegisterSubject = "login.register"
	LoginGetUserSubject  = "login.get_user"

	// ============ 游戏服务相关 ============
	GamePlayerRegisterSubject   = "game.player.register"
	GamePlayerUnregisterSubject = "game.player.unregister"
	GameStartSequenceSubject    = "game.sequence.start"
	GameStopSequenceSubject     = "game.sequence.stop"
	GameSequenceResultSubject   = "game.sequence.result"
	GamePlayerDataSubject       = "game.player.data"
	GameInventoryUpdateSubject  = "game.inventory.update"
	GameEquipmentUpdateSubject  = "game.equipment.update"

	// ============ 持久化服务相关 ============
	PersistSaveSubject       = "persist.save"
	PersistLoadSubject       = "persist.load"
	PersistLoadResultSubject = "persist.load_result"

	// ============ 网关服务相关 ============
	GatewayBroadcastSubject = "gateway.broadcast"
	GatewayClientMsgSubject = "gateway.client_msg"

	// ============ 系统广播相关 ============
	SystemHeartbeatSubject = "system.heartbeat"
	SystemStatusSubject    = "system.status"
)
