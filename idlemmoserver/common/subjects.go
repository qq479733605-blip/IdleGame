package common

// NATS主题定义
const (
	// ============ 认证服务相关 ============
	AuthPasswordSubject      = "auth.password"
	AuthRegisterSubject      = "auth.register"
	AuthGetUserSubject       = "auth.get_user"
	AuthValidateTokenSubject = "auth.validate_token"

	// ============ OAuth服务相关 ============
	OAuthAuthURLSubject  = "oauth.auth_url"
	OAuthCallbackSubject = "oauth.callback"
	OAuthUserInfoSubject = "oauth.user_info"

	// ============ 统一对外认证主题 ============
	AuthLoginSubject     = "auth.login"      // Gateway调用
	AuthGetPlayerSubject = "auth.get_player" // 根据token获取playerID

	// ============ 游戏服务相关 ============
	GamePlayerConnectSubject    = "game.player.connect"
	GamePlayerDisconnectSubject = "game.player.disconnect"
	GamePlayerRegisterSubject   = "game.player.register"
	GamePlayerUnregisterSubject = "game.player.unregister"
	GameStateSubject            = "game.state"
	GameActionSubject           = "game.action"

	// ============ 持久化服务相关 ============
	PersistSaveSubject       = "persist.save"
	PersistLoadSubject       = "persist.load"
	PersistLoadResultSubject = "persist.load_result"

	// ============ 用户数据持久化相关 ============
	PersistSaveUserSubject   = "persist.save_user"
	PersistLoadUserSubject   = "persist.load_user"
	PersistUserExistsSubject = "persist.user_exists"
	PersistSavePlayerSubject = "persist.save_player"
	PersistLoadPlayerSubject = "persist.load_player"

	// ============ 网关服务相关 ============
	GatewayBroadcastSubject = "gateway.broadcast"
	GatewayClientMsgSubject = "gateway.client_msg"

	// ============ 系统广播相关 ============
	SystemHeartbeatSubject = "system.heartbeat"
	SystemStatusSubject    = "system.status"
)
