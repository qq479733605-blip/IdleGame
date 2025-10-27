package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// Service 网关服务
type Service struct {
	nc          *nats.Conn
	system      *actor.ActorSystem
	gatewayPID  *actor.PID
	upgrader    WebSocketUpgrader
	connections map[string]*ClientConnection
}

// NewService 创建新的网关服务
func NewService() *Service {
	return &Service{
		upgrader:    NewWebSocketUpgrader(),
		connections: make(map[string]*ClientConnection),
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	// 连接NATS
	log.Printf("Connecting to NATS at %s", common.NATSURL)
	nc, err := nats.Connect(common.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	s.nc = nc
	log.Printf("Successfully connected to NATS")

	// 创建Actor系统
	s.system = actor.NewActorSystem()

	// 创建并启动网关Actor，传入NATS连接
	props := actor.PropsFromProducer(func() actor.Actor {
		return &GatewayActor{
			connections: s.connections,
			nc:          s.nc,
		}
	})
	s.gatewayPID = s.system.Root.Spawn(props)

	// 注册NATS处理器
	if err := s.registerNATSHandlers(); err != nil {
		return fmt.Errorf("failed to register NATS handlers: %w", err)
	}

	log.Printf("Gateway service started successfully")
	return nil
}

// Stop 停止服务
func (s *Service) Stop(ctx context.Context) error {
	// 关闭所有连接
	for _, conn := range s.connections {
		conn.Close()
	}

	if s.nc != nil {
		s.nc.Close()
	}
	if s.system != nil {
		s.system.Shutdown()
	}
	return nil
}

// GetHTTPHandler 获取HTTP处理器
func (s *Service) GetHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// WebSocket升级端点
	mux.HandleFunc("/ws", s.handleWebSocket)

	// 健康检查端点
	mux.HandleFunc("/health", s.handleHealth)

	// 调试端点
	mux.HandleFunc("/debug", s.handleDebug)

	// 认证端点 - 转发到Login服务
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/register", s.handleRegister)

	// 应用CORS中间件
	return common.NewCORSMiddleware(mux)
}

// handleWebSocket 处理WebSocket连接
func (s *Service) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== WebSocket Connection Request ===")
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Headers:")
	for name, values := range r.Header {
		log.Printf("  %s: %v", name, values)
	}
	log.Printf("Query params: %s", r.URL.RawQuery)

	// 升级到WebSocket连接
	conn, err := s.upgrader.Upgrade(w, r)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	log.Printf("WebSocket connection established from: %s", conn.RemoteAddr().String())

	// 创建客户端连接
	clientConn := NewClientConnection(conn, s.system, s.gatewayPID, s.nc)
	log.Printf("Created ClientConnection for %s", conn.RemoteAddr().String())

	// 启动连接处理
	log.Printf("Starting ClientConnection...")
	go clientConn.Start()
	log.Printf("ClientConnection started in goroutine")

	// 添加到连接管理器
	s.connections[conn.RemoteAddr().String()] = clientConn
	log.Printf("WebSocket connection added to manager")
	log.Printf("===============================")
}

// handleHealth 处理健康检查
func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// registerNATSHandlers 注册NATS处理器
func (s *Service) registerNATSHandlers() error {
	// 注册广播处理器
	broadcastSub, err := s.nc.Subscribe(common.GatewayBroadcastSubject, func(msg *nats.Msg) {
		s.handleBroadcast(msg)
	})
	if err != nil {
		return err
	}

	// 保存订阅以便清理
	go func() {
		<-context.Background().Done()
		broadcastSub.Unsubscribe()
	}()

	return nil
}

// handleBroadcast 处理广播消息
func (s *Service) handleBroadcast(msg *nats.Msg) {
	var broadcast common.MsgToClient
	if err := common.Unmarshal(msg.Data, &broadcast); err != nil {
		log.Printf("Failed to unmarshal broadcast message: %v", err)
		return
	}

	// 发送给对应的客户端
	for _, conn := range s.connections {
		if conn.playerID == broadcast.PlayerID {
			conn.Send(broadcast.Data)
			break
		}
	}
}

// handleLogin 处理登录请求 - 通过NATS调用Auth服务
func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== LOGIN REQUEST ===")
	log.Printf("Method: %s", r.Method)
	log.Printf("URL: %s", r.URL.String())
	log.Printf("Headers:")
	for name, values := range r.Header {
		log.Printf("  %s: %v", name, values)
	}

	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode login request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Login request for user: %s", req.Username)
	log.Printf("===================")

	// 创建认证请求消息
	authMsg := &common.MsgAuthenticateUser{
		Username: req.Username,
		Password: req.Password,
		ReplyTo:  nil, // 将使用Request模式
	}

	// 通过NATS发送请求到Auth服务
	authData, err := common.Marshal(authMsg)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}

	response, err := s.nc.Request(common.AuthLoginSubject, authData, 5*time.Second)
	if err != nil {
		log.Printf("Failed to call auth service: %v", err)
		http.Error(w, "Auth service unavailable", http.StatusBadGateway)
		return
	}

	// 解析响应
	var result common.MsgAuthenticateUserResult
	if err := common.Unmarshal(response.Data, &result); err != nil {
		log.Printf("Failed to parse auth response: %v", err)
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
	json.NewEncoder(w).Encode(result)
}

// handleRegister 处理注册请求 - 通过NATS调用Auth服务
func (s *Service) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 创建注册请求消息
	regMsg := &common.MsgRegisterUser{
		Username: req.Username,
		Password: req.Password,
		ReplyTo:  nil, // 将使用Request模式
	}

	// 通过NATS发送请求
	regData, err := common.Marshal(regMsg)
	if err != nil {
		http.Error(w, "Failed to marshal request", http.StatusInternalServerError)
		return
	}
	response, err := s.nc.Request(common.AuthRegisterSubject, regData, 5*time.Second)
	if err != nil {
		log.Printf("Failed to call auth service: %v", err)
		http.Error(w, "Auth service unavailable", http.StatusBadGateway)
		return
	}

	// 解析响应
	var result common.MsgRegisterUserResult
	if err := common.Unmarshal(response.Data, &result); err != nil {
		log.Printf("Failed to parse register response: %v", err)
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(result)
}
