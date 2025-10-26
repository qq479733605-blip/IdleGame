package login

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// Service 登录服务
type Service struct {
	nc       *nats.Conn
	system   *actor.ActorSystem
	userRepo common.UserRepository
	loginPID *actor.PID
	server   *http.Server
}

// NewService 创建新的登录服务
func NewService() *Service {
	return &Service{
		userRepo: NewMemoryUserRepository(),
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	// 连接NATS
	nc, err := nats.Connect(common.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	s.nc = nc

	// 创建Actor系统
	s.system = actor.NewActorSystem()

	// 创建并启动登录Actor
	props := actor.PropsFromProducer(NewLoginActor(s.userRepo))
	s.loginPID = s.system.Root.Spawn(props)

	// 注册NATS处理器
	if err := s.registerNATSHandlers(s.loginPID); err != nil {
		return fmt.Errorf("failed to register NATS handlers: %w", err)
	}

	// Login服务现在只通过NATS通信，不启动HTTP服务器
	log.Printf("Login service started successfully (NATS only)")
	return nil
}

// Stop 停止服务
func (s *Service) Stop(ctx context.Context) error {
	if s.nc != nil {
		s.nc.Close()
	}
	if s.system != nil {
		s.system.Shutdown()
	}
	return nil
}

// registerNATSHandlers 注册NATS处理器
func (s *Service) registerNATSHandlers(loginPID *actor.PID) error {
	// 注册认证处理器
	authSub, err := s.nc.Subscribe(common.LoginAuthSubject, func(msg *nats.Msg) {
		s.handleAuth(loginPID, msg)
	})
	if err != nil {
		return err
	}

	// 注册用户注册处理器
	regSub, err := s.nc.Subscribe(common.LoginRegisterSubject, func(msg *nats.Msg) {
		s.handleRegister(loginPID, msg)
	})
	if err != nil {
		return err
	}

	// 注册获取用户处理器
	getUserSub, err := s.nc.Subscribe(common.LoginGetUserSubject, func(msg *nats.Msg) {
		s.handleGetUser(loginPID, msg)
	})
	if err != nil {
		return err
	}

	// 保存订阅以便清理 - 暂时不处理清理
	// go func() {
	// 	<-ctx.Done()
	// 	authSub.Unsubscribe()
	// 	regSub.Unsubscribe()
	// 	getUserSub.Unsubscribe()
	// }()

	// 使用变量避免编译错误
	_ = authSub
	_ = regSub
	_ = getUserSub

	log.Printf("NATS handlers registered for login service")
	return nil
}

// handleAuth 处理认证请求
func (s *Service) handleAuth(loginPID *actor.PID, msg *nats.Msg) {
	var req common.MsgAuthenticateUser
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal auth request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &AuthReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	req.ReplyTo = replyPID
	s.system.Root.Send(loginPID, &req)
}

// handleRegister 处理注册请求
func (s *Service) handleRegister(loginPID *actor.PID, msg *nats.Msg) {
	var req common.MsgRegisterUser
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal register request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &RegisterReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	req.ReplyTo = replyPID
	s.system.Root.Send(loginPID, &req)
}

// handleGetUser 处理获取用户请求
func (s *Service) handleGetUser(loginPID *actor.PID, msg *nats.Msg) {
	var req common.MsgGetUserByPlayerID
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal get user request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &GetUserReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	req.ReplyTo = replyPID
	s.system.Root.Send(loginPID, &req)
}

// startHTTPServer 启动HTTP服务器
func (s *Service) startHTTPServer() {
	mux := http.NewServeMux()

	// 创建简单的HTTP处理器
	simpleHandler := NewSimpleHTTPHandler(s.userRepo)

	// 登录接口 - 使用简单处理器
	mux.HandleFunc("/login", simpleHandler.HandleSimpleLogin)

	// 注册接口 - 使用简单处理器
	mux.HandleFunc("/register", simpleHandler.HandleSimpleRegister)

	// 健康检查
	mux.HandleFunc("/health", s.handleHealth)

	// 应用CORS中间件
	handlerWithCORS := common.NewCORSMiddleware(mux)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", common.LoginServicePort),
		Handler: handlerWithCORS,
	}

	// 在goroutine中启动HTTP服务器
	go func() {
		log.Printf("Login service HTTP server listening on port %d", common.LoginServicePort)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
}

// handleLoginHTTP 处理HTTP登录请求
func (s *Service) handleLoginHTTP(w http.ResponseWriter, r *http.Request) {
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

	// 创建认证请求消息
	authMsg := &common.MsgAuthenticateUser{
		Username: req.Username,
		Password: req.Password,
		ReplyTo:  nil, // 将在下面的reply handler中处理
	}

	// 创建一个channel来等待响应
	responseChan := make(chan interface{}, 1)

	// 创建临时回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &HTTPAuthReplyActor{responseChan: responseChan}
	})
	replyPID := s.system.Root.Spawn(replyProps)
	defer s.system.Root.Stop(replyPID)

	authMsg.ReplyTo = replyPID

	// 发送认证请求到LoginActor
	s.system.Root.Send(s.loginPID, authMsg)

	// 等待响应
	select {
	case response := <-responseChan:
		switch resp := response.(type) {
		case *common.MsgAuthenticateUserResult:
			if resp.Success {
				response := map[string]interface{}{
					"success":  true,
					"token":    resp.Token,
					"playerID": resp.PlayerID,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			} else {
				response := map[string]interface{}{
					"success": false,
					"error":   resp.Message,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(response)
			}
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	case <-time.After(5 * time.Second):
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleRegisterHTTP 处理HTTP注册请求
func (s *Service) handleRegisterHTTP(w http.ResponseWriter, r *http.Request) {
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
		ReplyTo:  nil,
	}

	// 创建一个channel来等待响应
	responseChan := make(chan interface{}, 1)

	// 创建临时回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &HTTPRegisterReplyActor{responseChan: responseChan}
	})
	replyPID := s.system.Root.Spawn(replyProps)
	defer s.system.Root.Stop(replyPID)

	regMsg.ReplyTo = replyPID

	// 发送注册请求到LoginActor
	s.system.Root.Send(s.loginPID, regMsg)

	// 等待响应
	select {
	case response := <-responseChan:
		switch resp := response.(type) {
		case *common.MsgRegisterUserResult:
			if resp.Success {
				response := map[string]interface{}{
					"success": true,
					"message": "User registered successfully",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			} else {
				response := map[string]interface{}{
					"success": false,
					"error":   resp.Message,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(response)
			}
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	case <-time.After(5 * time.Second):
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
	}
}

// handleHealth 处理健康检查
func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// GenerateToken 生成认证Token
func GenerateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
