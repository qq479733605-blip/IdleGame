package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/golang-jwt/jwt/v5"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// Service 认证服务
type Service struct {
	nc        *nats.Conn
	system    *actor.ActorSystem
	authPID   *actor.PID
	jwtSecret []byte
}

// NewService 创建新的认证服务
func NewService() *Service {
	return &Service{
		jwtSecret: []byte("your-secret-key-change-in-production"),
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

	// 创建并启动认证Actor
	props := actor.PropsFromProducer(NewAuthActor(s.jwtSecret, s.nc))
	s.authPID = s.system.Root.Spawn(props)

	// 注册NATS处理器
	if err := s.registerNATSHandlers(s.authPID); err != nil {
		return fmt.Errorf("failed to register NATS handlers: %w", err)
	}

	log.Printf("Auth service started successfully (NATS only)")
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
func (s *Service) registerNATSHandlers(authPID *actor.PID) error {
	// 统一登录处理器
	loginSub, err := s.nc.Subscribe(common.AuthLoginSubject, func(msg *nats.Msg) {
		s.handleLogin(authPID, msg)
	})
	if err != nil {
		return err
	}

	// 用户注册处理器
	regSub, err := s.nc.Subscribe(common.AuthRegisterSubject, func(msg *nats.Msg) {
		s.handleRegister(authPID, msg)
	})
	if err != nil {
		return err
	}

	// 获取用户处理器
	getUserSub, err := s.nc.Subscribe(common.AuthGetUserSubject, func(msg *nats.Msg) {
		s.handleGetUser(authPID, msg)
	})
	if err != nil {
		return err
	}

	// Token验证处理器
	validateTokenSub, err := s.nc.Subscribe(common.AuthValidateTokenSubject, func(msg *nats.Msg) {
		s.handleValidateToken(authPID, msg)
	})
	if err != nil {
		return err
	}

	// 根据Token获取PlayerID
	getPlayerSub, err := s.nc.Subscribe(common.AuthGetPlayerSubject, func(msg *nats.Msg) {
		s.handleGetPlayerByToken(authPID, msg)
	})
	if err != nil {
		return err
	}

	// 使用变量避免编译错误
	_ = loginSub
	_ = regSub
	_ = getUserSub
	_ = validateTokenSub
	_ = getPlayerSub

	log.Printf("NATS handlers registered for auth service")
	return nil
}

// handleLogin 处理登录请求（统一入口）
func (s *Service) handleLogin(authPID *actor.PID, msg *nats.Msg) {
	var req common.MsgAuthenticateUser
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal login request: %v", err)
		return
	}

	// 直接在服务中处理，避免Actor间通信问题
	result := s.processLogin(&req)

	// 序列化回复
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal login result: %v", err)
		return
	}

	// 直接回复NATS消息
	if err := msg.Respond(data); err != nil {
		log.Printf("Failed to respond to login request: %v", err)
	}
}

// handleRegister 处理注册请求
func (s *Service) handleRegister(authPID *actor.PID, msg *nats.Msg) {
	log.Printf("🔥 Auth service received register request via NATS!")
	log.Printf("🔥 Request data: %s", string(msg.Data))

	var req common.MsgRegisterUser
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal register request: %v", err)
		return
	}

	log.Printf("🔥 Successfully unmarshaled register request for user: %s", req.Username)

	// 直接在服务中处理，避免Actor间通信问题
	result := s.processRegister(&req)

	// 序列化回复
	data, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal register result: %v", err)
		return
	}

	// 直接回复NATS消息
	if err := msg.Respond(data); err != nil {
		log.Printf("Failed to respond to register request: %v", err)
	}
}

// handleGetUser 处理获取用户请求
func (s *Service) handleGetUser(authPID *actor.PID, msg *nats.Msg) {
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
	s.system.Root.Send(authPID, &req)
}

// handleValidateToken 处理Token验证请求
func (s *Service) handleValidateToken(authPID *actor.PID, msg *nats.Msg) {
	var req common.MsgValidateToken
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal validate token request: %v", err)
		return
	}

	// 直接处理token验证，不使用Actor系统
	claims, err := s.validateJWT(req.Token)
	if err != nil {
		result := &common.MsgValidateTokenResult{
			Valid:   false,
			Message: err.Error(),
		}
		data, _ := json.Marshal(result)
		msg.Respond(data)
		return
	}

	playerID, ok := (*claims)["playerID"].(string)
	if !ok {
		result := &common.MsgValidateTokenResult{
			Valid:   false,
			Message: "invalid player ID in token",
		}
		data, _ := json.Marshal(result)
		msg.Respond(data)
		return
	}

	result := &common.MsgValidateTokenResult{
		Valid:    true,
		PlayerID: playerID,
	}
	data, _ := json.Marshal(result)
	msg.Respond(data)
}

// validateJWT 验证JWT令牌
func (s *Service) validateJWT(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// handleGetPlayerByToken 处理根据Token获取PlayerID请求
func (s *Service) handleGetPlayerByToken(authPID *actor.PID, msg *nats.Msg) {
	var req common.MsgGetPlayerByToken
	if err := common.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal get player by token request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &GetPlayerByTokenReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	req.ReplyTo = replyPID
	s.system.Root.Send(authPID, &req)
}

// GenerateToken 生成JWT Token
func (s *Service) GenerateToken(playerID string) (string, error) {
	claims := jwt.MapClaims{
		"playerID": playerID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken 验证JWT Token
func (s *Service) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// processLogin 处理登录逻辑
func (s *Service) processLogin(req *common.MsgAuthenticateUser) *common.MsgAuthenticateUserResult {
	// 通过NATS请求加载用户数据
	loadMsg := &common.MsgLoadUser{
		Username: req.Username,
	}

	data, err := json.Marshal(loadMsg)
	if err != nil {
		log.Printf("Failed to marshal load user message: %v", err)
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "加载用户数据失败",
		}
	}

	// 发送NATS请求到Persist服务
	resp, err := s.nc.Request(common.PersistLoadUserSubject, data, 5*time.Second)
	if err != nil {
		log.Printf("Failed to load user via NATS: %v", err)
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "无法连接到持久化服务",
		}
	}

	var loadResult common.MsgLoadUserResult
	if err := json.Unmarshal(resp.Data, &loadResult); err != nil {
		log.Printf("Failed to unmarshal load user result: %v", err)
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "解析用户数据失败",
		}
	}

	if loadResult.Err != nil {
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "用户不存在",
		}
	}

	user := loadResult.UserData

	// 验证密码
	if !verifyPassword(req.Password, user.Password) {
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "密码错误",
		}
	}

	// 更新最后登录时间
	user.LastLogin = time.Now()

	// 通过NATS保存更新的用户数据
	saveMsg := &common.MsgSaveUser{
		UserData: user,
	}

	saveData, err := json.Marshal(saveMsg)
	if err != nil {
		log.Printf("Failed to marshal save user message: %v", err)
	} else {
		if _, err := s.nc.Request(common.PersistSaveUserSubject, saveData, 5*time.Second); err != nil {
			log.Printf("Failed to save user via NATS: %v", err)
		}
	}

	// 生成JWT Token
	token, err := s.GenerateToken(user.PlayerID)
	if err != nil {
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "生成token失败",
		}
	}

	return &common.MsgAuthenticateUserResult{
		Success:  true,
		Message:  "登录成功",
		PlayerID: user.PlayerID,
		Token:    token,
	}
}

// processRegister 处理注册逻辑
func (s *Service) processRegister(req *common.MsgRegisterUser) *common.MsgRegisterUserResult {
	// 通过NATS检查用户是否已存在
	existsMsg := &common.MsgUserExists{
		Username: req.Username,
	}

	data, err := json.Marshal(existsMsg)
	if err != nil {
		log.Printf("Failed to marshal user exists message: %v", err)
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 序列化请求出错",
		}
	}

	// 发送NATS请求到Persist服务
	resp, err := s.nc.Request(common.PersistUserExistsSubject, data, 5*time.Second)
	if err != nil {
		log.Printf("Failed to check user existence via NATS: %v", err)
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 无法连接到持久化服务",
		}
	}

	var existsResult common.MsgUserExistsResult
	if err := json.Unmarshal(resp.Data, &existsResult); err != nil {
		log.Printf("Failed to unmarshal user exists result: %v", err)
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 解析响应出错",
		}
	}

	// 如果用户已存在，返回错误
	if existsResult.Exists {
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "用户名已存在",
		}
	}

	// 生成PlayerID
	playerID := GenerateRandomString(16)

	// 创建用户
	user := &common.UserData{
		Username:  req.Username,
		Password:  hashPassword(req.Password),
		PlayerID:  playerID,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}

	// 通过NATS保存用户数据
	saveMsg := &common.MsgSaveUser{
		UserData: user,
	}

	saveData, err := json.Marshal(saveMsg)
	if err != nil {
		log.Printf("Failed to marshal save user message: %v", err)
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 序列化用户数据出错",
		}
	}

	// 发送NATS请求到Persist服务
	_, err = s.nc.Request(common.PersistSaveUserSubject, saveData, 5*time.Second)
	if err != nil {
		log.Printf("Failed to save user via NATS: %v", err)
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 无法保存用户数据",
		}
	}

	return &common.MsgRegisterUserResult{
		Success:  true,
		Message:  "注册成功",
		PlayerID: playerID,
	}
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}
