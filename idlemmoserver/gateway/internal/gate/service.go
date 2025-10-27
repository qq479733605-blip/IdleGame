package gate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/idle-server/common"
	"github.com/idle-server/common/nats"
	natsio "github.com/nats-io/nats.go"
)

// Service 网关服务 - 使用 Gin + Gorilla WebSocket + NATS
type Service struct {
	natsManager *nats.Manager
	upgrader    WebSocketUpgrader
	connections sync.Map // map[string]*ClientConnection
	broadcastCh chan BroadcastMessage
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	PlayerID string
	Data     []byte
}

// NewService 创建新的网关服务
func NewService() *Service {
	return &Service{
		upgrader:    NewWebSocketUpgrader(),
		broadcastCh: make(chan BroadcastMessage, 1000),
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	// 连接NATS
	log.Printf("Connecting to NATS at %s", common.NATSURL)
	natsManager, err := nats.NewManager(common.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	s.natsManager = natsManager
	log.Printf("Successfully connected to NATS")

	// 注册NATS处理器
	if err := s.registerNATSHandlers(); err != nil {
		return fmt.Errorf("failed to register NATS handlers: %w", err)
	}

	// 启动广播处理器
	go s.broadcastWorker()

	log.Printf("Gateway service started successfully")
	return nil
}

// Stop 停止服务
func (s *Service) Stop(ctx context.Context) error {
	// 关闭所有连接
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*ClientConnection); ok {
			conn.Close()
		}
		return true
	})

	close(s.broadcastCh)

	if s.natsManager != nil {
		s.natsManager.Close()
	}
	return nil
}

// GetHTTPHandler 获取 Gin HTTP 处理器
func (s *Service) GetHTTPHandler() *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// 添加中间件
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(s.corsMiddleware())

	// WebSocket 升级端点
	r.GET("/ws", s.handleWebSocket)

	// 认证端点（保持原有路径）
	r.POST("/login", s.handleLogin)
	r.POST("/register", s.handleRegister)

	// 健康检查端点
	r.GET("/health", s.handleHealth)
	r.GET("/debug", s.handleDebug)

	return r
}

// handleWebSocket 处理 WebSocket 连接
func (s *Service) handleWebSocket(c *gin.Context) {
	log.Printf("=== WebSocket Connection Request ===")
	log.Printf("URL: %s", c.Request.URL.String())
	log.Printf("Headers:")
	for name, values := range c.Request.Header {
		log.Printf("  %s: %v", name, values)
	}
	log.Printf("Query params: %s", c.Request.URL.RawQuery)

	// 升级到 WebSocket 连接
	conn, err := s.upgrader.Upgrade(c.Writer, c.Request)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}

	log.Printf("WebSocket connection established from: %s", conn.RemoteAddr().String())

	// 创建客户端连接
	clientConn := NewClientConnection(conn, s, s.onConnectionClose)

	// 启动连接处理
	go clientConn.Start()

	// 添加到连接管理器
	s.connections.Store(conn.RemoteAddr().String(), clientConn)
	log.Printf("WebSocket connection added to manager")
	log.Printf("===============================")
}

// handleHealth 处理健康检查
func (s *Service) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"service":   "gateway",
		"timestamp": time.Now().Unix(),
	})
}

// handleDebug 处理调试信息
func (s *Service) handleDebug(c *gin.Context) {
	connections := make([]gin.H, 0)
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*ClientConnection); ok {
			connections = append(connections, gin.H{
				"addr":     key.(string),
				"playerID": conn.GetPlayerID(),
			})
		}
		return true
	})

	natsStatus := "disconnected"
	if s.natsManager != nil && s.natsManager.IsConnected() {
		natsStatus = "connected"
	}

	c.JSON(http.StatusOK, gin.H{
		"connections": connections,
		"total":       len(connections),
		"nats_status": natsStatus,
	})
}

// registerNATSHandlers 注册 NATS 处理器
func (s *Service) registerNATSHandlers() error {
	// 订阅广播消息
	broadcastHandler := &broadcastMessageHandler{
		broadcastCh: s.broadcastCh,
	}

	_, err := s.natsManager.Subscribe(common.GatewayBroadcastSubject, broadcastHandler)
	return err
}

// broadcastWorker 广播消息处理器
func (s *Service) broadcastWorker() {
	for msg := range s.broadcastCh {
		// 发送给对应的客户端
		s.connections.Range(func(key, value interface{}) bool {
			if conn, ok := value.(*ClientConnection); ok {
				if conn.GetPlayerID() == msg.PlayerID {
					conn.Send(msg.Data)
					return false // 找到目标后停止遍历
				}
			}
			return true
		})
	}
}

// handleLogin 处理登录请求
func (s *Service) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Login request for user: %s", req.Username)

	result, err := s.authenticateUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Failed to authenticate user: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Auth service unavailable"})
		return
	}

	// 返回结果
	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusUnauthorized, result)
	}
}

// handleRegister 处理注册请求
func (s *Service) handleRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.registerUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Failed to register user: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Auth service unavailable"})
		return
	}

	// 返回结果
	if result.Success {
		c.JSON(http.StatusCreated, result)
	} else {
		c.JSON(http.StatusBadRequest, result)
	}
}

// HandleMessage 实现 MessageHandler 接口 - 处理来自 WebSocket 连接的消息
func (s *Service) HandleMessage(playerID string, data []byte) error {
	// 解析客户端消息
	var clientMsg map[string]interface{}
	if err := json.Unmarshal(data, &clientMsg); err != nil {
		return err
	}

	msgType, ok := clientMsg["type"].(string)
	if !ok {
		return fmt.Errorf("missing message type")
	}

	log.Printf("Processing message type: %s from player: %s", msgType, playerID)

	switch msgType {
	case "C_Login":
		return s.handleWSLogin(playerID, data)
	case "C_Ping":
		return s.handleWSPing(playerID)
	case "C_ClientPayload":
		return s.handleWSClientPayload(playerID, data)
	default:
		log.Printf("Unknown message type: %s", msgType)
		return fmt.Errorf("unknown message type: %s", msgType)
	}
}

// onConnectionClose 连接关闭回调
func (s *Service) onConnectionClose(playerID string) {
	log.Printf("Connection closed for player: %s", playerID)
	// 从连接管理器中移除连接
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*ClientConnection); ok {
			if conn.GetPlayerID() == playerID {
				s.connections.Delete(key)
				return false // 找到后停止遍历
			}
		}
		return true
	})
}

// handleWSLogin 处理 WebSocket 登录消息
func (s *Service) handleWSLogin(playerID string, data []byte) error {
	var loginMsg struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(data, &loginMsg); err != nil {
		return err
	}

	// 验证 token 并获取 playerID
	extractedPlayerID, err := s.extractPlayerIDFromToken(loginMsg.Token)
	if err != nil {
		log.Printf("Failed to extract playerID from token: %v", err)
		s.sendToConnection(playerID, s.createErrorMessage("Invalid token"))
		return err
	}

	// 注册玩家到 Game 服务
	if err := s.registerPlayerToGame(extractedPlayerID); err != nil {
		log.Printf("Failed to register player to game service: %v", err)
		s.sendToConnection(playerID, s.createErrorMessage("Failed to register player"))
		return err
	}

	// 更新连接的 playerID
	s.updateConnectionPlayerID(playerID, extractedPlayerID)

	// 发送登录成功消息
	s.sendToConnection(extractedPlayerID, s.createLoginSuccessMessage(extractedPlayerID))
	log.Printf("Player %s logged in and registered to game service", extractedPlayerID)

	return nil
}

// handleWSPing 处理 WebSocket ping 消息
func (s *Service) handleWSPing(playerID string) error {
	pongMsg := map[string]interface{}{
		"type": "S_Pong",
		"time": time.Now().Unix(),
	}
	data, _ := json.Marshal(pongMsg)
	s.sendToConnection(playerID, data)
	return nil
}

// handleWSClientPayload 处理 WebSocket 客户端业务消息
func (s *Service) handleWSClientPayload(playerID string, data []byte) error {
	if playerID == "" {
		return fmt.Errorf("player not authenticated")
	}

	// 解析客户端消息
	var clientMsg map[string]interface{}
	if err := json.Unmarshal(data, &clientMsg); err != nil {
		return err
	}

	msgType, ok := clientMsg["type"].(string)
	if !ok {
		return fmt.Errorf("missing message type")
	}

	log.Printf("Processing client message type: %s from player %s", msgType, playerID)

	// 这里可以转发给相应的游戏服务处理
	// 目前只是记录日志
	log.Printf("Game message received: %s", msgType)
	return nil
}

// 辅助方法
func (s *Service) extractPlayerIDFromToken(tokenString string) (string, error) {
	// 简化的 JWT 解析实现
	// 在生产环境中应该向 Auth 服务验证
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	// 这里应该实现完整的 JWT 解析和验证
	// 为了简化，暂时返回一个假的 playerID
	return "temp_player_id", nil
}

func (s *Service) updateConnectionPlayerID(oldPlayerID, newPlayerID string) {
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*ClientConnection); ok {
			if conn.GetPlayerID() == oldPlayerID {
				conn.SetPlayerID(newPlayerID)
				return false
			}
		}
		return true
	})
}

func (s *Service) sendToConnection(playerID string, data []byte) {
	s.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*ClientConnection); ok {
			if conn.GetPlayerID() == playerID {
				conn.Send(data)
				return false
			}
		}
		return true
	})
}

func (s *Service) createErrorMessage(message string) []byte {
	response := map[string]interface{}{
		"type":  "S_Error",
		"error": message,
	}
	data, _ := json.Marshal(response)
	return data
}

func (s *Service) createLoginSuccessMessage(playerID string) []byte {
	response := map[string]interface{}{
		"type":      "S_LoginOK",
		"token":     "", // 由 Game 服务填充
		"player_id": playerID,
	}
	data, _ := json.Marshal(response)
	return data
}

// corsMiddleware CORS 中间件
func (s *Service) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 允许的来源
		allowedOrigins := []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:3000",
		}

		// 检查来源是否在允许列表中
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// WebSocketUpgrader WebSocket升级器
type WebSocketUpgrader struct {
	upgrader websocket.Upgrader
}

// NewWebSocketUpgrader 创建WebSocket升级器
func NewWebSocketUpgrader() WebSocketUpgrader {
	return WebSocketUpgrader{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// 在生产环境中应该检查Origin
				return true
			},
			// 禁用压缩扩展以避免兼容性问题
			EnableCompression: false,
		},
	}
}

// Upgrade 升级HTTP连接到WebSocket
func (u *WebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return u.upgrader.Upgrade(w, r, nil)
}

// broadcastMessageHandler 广播消息处理器
type broadcastMessageHandler struct {
	broadcastCh chan<- BroadcastMessage
}

// Handle 实现MessageHandler接口
func (h *broadcastMessageHandler) Handle(msg *natsio.Msg) error {
	var broadcast common.MsgToClient
	if err := common.Unmarshal(msg.Data, &broadcast); err != nil {
		log.Printf("Failed to unmarshal broadcast message: %v", err)
		return err
	}

	// 通过channel发送给广播处理器
	select {
	case h.broadcastCh <- BroadcastMessage{
		PlayerID: broadcast.PlayerID,
		Data:     broadcast.Data,
	}:
	default:
		log.Printf("Broadcast channel is full, dropping message")
	}

	return nil
}

// 辅助方法 - 使用统一的NATS管理器

// authenticateUser 认证用户
func (s *Service) authenticateUser(username, password string) (*common.MsgAuthenticateUserResult, error) {
	authMsg := map[string]interface{}{
		"type":     "C_Login",
		"username": username,
		"password": password,
	}

	response, err := s.natsManager.Request(common.AuthLoginSubject, authMsg, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to call auth service: %w", err)
	}

	// 响应可能是两种格式：
	// 1. 统一的 Response 格式
	// 2. base64 编码的旧格式结果

	// 首先尝试作为统一 Response 格式解析
	var handlerResponse struct {
		Success bool        `json:"success"`
		Data    interface{} `json:"data,omitempty"`
		Error   string      `json:"error,omitempty"`
	}

	if err := common.Unmarshal(response.Data, &handlerResponse); err == nil {
		if handlerResponse.Success && handlerResponse.Data != nil {
			// 成功解析为统一格式，从 Data 中提取结果
			dataBytes, err := json.Marshal(handlerResponse.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response data: %w", err)
			}

			var result common.MsgAuthenticateUserResult
			if err := json.Unmarshal(dataBytes, &result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal auth result: %w", err)
			}

			return &result, nil
		} else if !handlerResponse.Success {
			// 处理错误响应 - 将错误信息转换为认证失败结果
			return &common.MsgAuthenticateUserResult{
				Success: false,
				Message: handlerResponse.Error,
			}, nil
		}
	}

	// 如果统一格式解析失败，尝试作为 base64 编码的旧格式解析
	var responseStr string
	if err := json.Unmarshal(response.Data, &responseStr); err == nil {
		// 解码 base64
		decoded, err := base64.StdEncoding.DecodeString(responseStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 response: %w", err)
		}

		var result common.MsgAuthenticateUserResult
		if err := json.Unmarshal(decoded, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal decoded auth result: %w", err)
		}

		return &result, nil
	}

	// 如果两种格式都解析失败，返回错误
	return nil, fmt.Errorf("failed to parse handler response: unknown format")
}

// registerUser 注册用户
func (s *Service) registerUser(username, password string) (*common.MsgRegisterUserResult, error) {
	regMsg := map[string]interface{}{
		"type":     "C_Register",
		"username": username,
		"password": password,
	}

	response, err := s.natsManager.Request(common.AuthRegisterSubject, regMsg, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to call auth service: %w", err)
	}

	// 响应可能是两种格式：
	// 1. 统一的 Response 格式
	// 2. base64 编码的旧格式结果

	// 首先尝试作为统一 Response 格式解析
	var handlerResponse struct {
		Success bool        `json:"success"`
		Data    interface{} `json:"data,omitempty"`
		Error   string      `json:"error,omitempty"`
	}

	if err := common.Unmarshal(response.Data, &handlerResponse); err == nil {
		if handlerResponse.Success && handlerResponse.Data != nil {
			// 成功解析为统一格式，从 Data 中提取结果
			dataBytes, err := json.Marshal(handlerResponse.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal response data: %w", err)
			}

			var result common.MsgRegisterUserResult
			if err := json.Unmarshal(dataBytes, &result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal register result: %w", err)
			}

			return &result, nil
		} else if !handlerResponse.Success {
			// 处理错误响应 - 将错误信息转换为注册失败结果
			return &common.MsgRegisterUserResult{
				Success: false,
				Message: handlerResponse.Error,
			}, nil
		}
	}

	// 如果统一格式解析失败，尝试作为 base64 编码的旧格式解析
	var responseStr string
	if err := json.Unmarshal(response.Data, &responseStr); err == nil {
		// 解码 base64
		decoded, err := base64.StdEncoding.DecodeString(responseStr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 response: %w", err)
		}

		var result common.MsgRegisterUserResult
		if err := json.Unmarshal(decoded, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal decoded register result: %w", err)
		}

		return &result, nil
	}

	// 如果两种格式都解析失败，返回错误
	return nil, fmt.Errorf("failed to parse handler response: unknown format")
}

// registerPlayerToGame 向游戏服务注册玩家
func (s *Service) registerPlayerToGame(playerID string) error {
	playerConnectMsg := &common.MsgPlayerConnect{
		PlayerID: playerID,
	}

	connectData, err := common.Marshal(playerConnectMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal player connect message: %w", err)
	}

	return s.natsManager.Publish(common.GamePlayerConnectSubject, connectData)
}
