package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/idle-server/common"
	"github.com/idle-server/common/database"
	"github.com/idle-server/common/handler"
	"github.com/idle-server/common/nats"
	"github.com/idle-server/common/service"
	natsio "github.com/nats-io/nats.go"
	"golang.org/x/crypto/bcrypt"
)

// Service 统一的认证服务
type Service struct {
	*service.BaseServiceImpl
	natsManager *nats.Manager
	processor   *handler.MessageProcessor
	jwtSecret   []byte
	gormDB      *database.GORM
	redis       *database.Redis
	userRepo    *database.GORMUserRepository
}

// NewService 创建新的认证服务
func NewService() service.Service {
	return &Service{
		BaseServiceImpl: service.NewBaseService("Auth"),
		jwtSecret:       []byte("your-secret-key-change-in-production"),
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	// 调用基类 Start 方法
	if err := s.BaseServiceImpl.Start(ctx); err != nil {
		return err
	}

	log.Println("Initializing Auth Service (Database + NATS)...")

	// 初始化GORM数据库
	gormConfig := database.DefaultGORMConfig()
	gormDB, err := database.NewGORM(gormConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize GORM database: %w", err)
	}
	s.gormDB = gormDB

	// 初始化Redis
	redisConfig := database.DefaultRedisConfig()
	redis, err := database.NewRedis(redisConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}
	s.redis = redis

	// 使用GORM的AutoMigrate功能运行数据库迁移
	if err := gormDB.AutoMigrate(
		&database.User{},
		&database.Player{},
		&database.GameProgress{},
	); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// 创建GORM仓库
	s.userRepo = database.NewGORMUserRepository(gormDB.GetDB(), redis)

	// 初始化 NATS 管理器
	s.natsManager, err = nats.NewManager(common.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to initialize NATS manager: %w", err)
	}

	// 初始化消息处理器
	s.processor = handler.NewMessageProcessor(s.natsManager)

	// 注册消息处理器
	if err := s.registerHandlers(); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	// 注册 NATS 订阅
	if err := s.registerNATSSubscriptions(); err != nil {
		return fmt.Errorf("failed to register NATS subscriptions: %w", err)
	}

	log.Printf("Auth Service started successfully with database and NATS")
	return nil
}

// Stop 停止服务
func (s *Service) Stop(ctx context.Context) error {
	// 调用基类 Stop 方法
	if err := s.BaseServiceImpl.Stop(ctx); err != nil {
		return err
	}

	log.Println("Stopping Auth Service...")

	// 关闭 NATS 管理器
	if s.natsManager != nil {
		s.natsManager.Close()
	}

	// 关闭Redis连接
	if s.redis != nil {
		if err := s.redis.Close(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}
	}

	// 关闭GORM数据库连接
	if s.gormDB != nil {
		if err := s.gormDB.Close(); err != nil {
			log.Printf("Error closing GORM database: %v", err)
		}
	}

	log.Println("Auth Service stopped successfully")
	return nil
}

// registerHandlers 注册消息处理器
func (s *Service) registerHandlers() error {
	// 注册登录处理器
	loginHandler := handler.NewLoginHandler(s.natsManager, s.authenticateUser)
	s.processor.RegisterHandler(loginHandler)

	// 注册注册处理器
	registerHandler := handler.NewRegisterHandler(s.natsManager, s.registerUser)
	s.processor.RegisterHandler(registerHandler)

	log.Printf("Auth handlers registered successfully")
	return nil
}

// registerNATSSubscriptions 注册 NATS 订阅
func (s *Service) registerNATSSubscriptions() error {
	// 使用统一的消息处理器订阅登录主题
	if _, err := s.natsManager.Subscribe(common.AuthLoginSubject, &natsMessageAdapter{
		processor: s.processor,
	}); err != nil {
		return fmt.Errorf("failed to subscribe to login subject: %w", err)
	}

	// 使用统一的消息处理器订阅注册主题
	if _, err := s.natsManager.Subscribe(common.AuthRegisterSubject, &natsMessageAdapter{
		processor: s.processor,
	}); err != nil {
		return fmt.Errorf("failed to subscribe to register subject: %w", err)
	}

	log.Printf("Auth NATS subscriptions registered successfully")
	return nil
}

// Auth Service 现在使用统一的 handler 框架
// 所有旧的消息处理代码已被移除，现在使用 common/handler 中的处理器

// natsMessageAdapter 消息处理器适配器
type natsMessageAdapter struct {
	processor *handler.MessageProcessor
}

// Handle 实现 nats.MessageHandler 接口
func (a *natsMessageAdapter) Handle(msg *natsio.Msg) error {
	return a.processor.ProcessMessage(msg)
}

// authenticateUser 认证用户业务逻辑
func (s *Service) authenticateUser(username, password string) (*common.MsgAuthenticateUserResult, error) {
	log.Printf("Processing login request for user: %s", username)

	// 检查用户是否存在
	userExists, err := s.checkUserExists(username)
	if err != nil {
		log.Printf("Failed to check user existence: %v", err)
		return nil, fmt.Errorf("authentication service error")
	}

	if !userExists {
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "User does not exist",
		}, nil
	}

	// 获取用户数据进行密码验证
	userData, err := s.getUserData(username)
	if err != nil {
		log.Printf("Failed to get user data: %v", err)
		return nil, fmt.Errorf("authentication service error")
	}

	// 使用bcrypt验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(password)); err != nil {
		log.Printf("Auth: Password verification failed for user %s: %v", userData.Username, err)
		return &common.MsgAuthenticateUserResult{
			Success: false,
			Message: "Invalid password",
		}, nil
	}
	log.Printf("Auth: Password verification successful for user %s", userData.Username)

	// 生成JWT令牌
	log.Printf("Auth: Generating JWT for PlayerID: %s", userData.PlayerID)
	token, err := s.generateJWT(userData.PlayerID)
	if err != nil {
		log.Printf("Failed to generate JWT: %v", err)
		return nil, fmt.Errorf("failed to generate authentication token")
	}
	log.Printf("Auth: JWT generated successfully: %s...", token[:50])

	result := &common.MsgAuthenticateUserResult{
		Success:  true,
		Message:  "Login successful",
		PlayerID: userData.PlayerID,
		Token:    token,
	}
	log.Printf("Auth: Login successful, returning result: Success=%t, PlayerID=%s", result.Success, result.PlayerID)
	return result, nil
}

// registerUser 注册用户业务逻辑
func (s *Service) registerUser(username, password string) (*common.MsgRegisterUserResult, error) {
	log.Printf("Processing registration request for user: %s", username)

	// 检查用户是否已存在
	userExists, err := s.checkUserExists(username)
	if err != nil {
		log.Printf("Failed to check user existence: %v", err)
		return nil, fmt.Errorf("registration service error")
	}

	if userExists {
		return &common.MsgRegisterUserResult{
			Success: false,
			Message: "Username already exists",
		}, nil
	}

	// 生成新的playerID
	playerID, err := s.generatePlayerID()
	if err != nil {
		log.Printf("Failed to generate player ID: %v", err)
		return nil, fmt.Errorf("registration service error")
	}

	// 创建用户数据
	userData := &common.UserData{
		Username:  username,
		Password:  password, // 实际应用中应该加密
		PlayerID:  playerID,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}

	// 保存用户数据
	if err := s.saveUserData(userData); err != nil {
		log.Printf("Failed to save user data: %v", err)
		return nil, fmt.Errorf("failed to create user account")
	}

	// 注册成功，记录日志
	log.Printf("User %s registered successfully with playerID: %s", username, playerID)

	// 注意：Token 将在登录接口中生成
	return &common.MsgRegisterUserResult{
		Success:  true,
		Message:  "Registration successful",
		PlayerID: playerID,
	}, nil
}

// ============ 辅助方法 ============

// generateJWT 生成JWT令牌
func (s *Service) generateJWT(playerID string) (string, error) {
	claims := jwt.MapClaims{
		"playerID": playerID,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 24小时过期
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// generatePlayerID 生成新的玩家ID
func (s *Service) generatePlayerID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "player_" + hex.EncodeToString(bytes), nil
}

// checkUserExists 检查用户是否存在
func (s *Service) checkUserExists(username string) (bool, error) {
	log.Printf("Auth: Checking if user exists in database: %s", username)

	// 使用GORM仓库直接查询数据库
	_, err := s.userRepo.GetUserByUsername(context.Background(), username)
	if err != nil {
		// 检查是否是"用户不存在"的错误
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "record not found") {
			log.Printf("Auth: User %s does not exist", username)
			return false, nil // 用户不存在是正常情况
		}
		log.Printf("Auth: Database error checking user %s: %v", username, err)
		return false, fmt.Errorf("database error checking user existence: %w", err)
	}

	log.Printf("Auth: User %s exists", username)
	return true, nil
}

// getUserData 获取用户数据
func (s *Service) getUserData(username string) (*common.UserData, error) {
	log.Printf("Auth: Loading user data from database: %s", username)

	// 使用GORM仓库直接从数据库加载用户数据
	userData, err := s.userRepo.GetUserByUsername(context.Background(), username)
	if err != nil {
		log.Printf("Auth: Failed to load user %s: %v", username, err)
		return nil, fmt.Errorf("failed to load user data: %w", err)
	}

	log.Printf("Auth: User data loaded successfully for: %s (PlayerID: %s)", userData.Username, userData.PlayerID)
	return userData, nil
}

// saveUserData 保存用户数据
func (s *Service) saveUserData(userData *common.UserData) error {
	log.Printf("Auth: Saving user data directly to database for user: %s", userData.Username)

	// 使用GORM仓库直接保存用户到数据库
	savedUserData, err := s.userRepo.CreateUser(context.Background(), userData.Username, userData.Password)
	if err != nil {
		log.Printf("Failed to save user %s: %v", userData.Username, err)
		return fmt.Errorf("failed to save user data: %w", err)
	}

	log.Printf("User data saved successfully for: %s (Generated PlayerID: %s)", savedUserData.Username, savedUserData.PlayerID)
	return nil
}

// Auth 服务重构完成
// 使用统一的架构：
// - 继承 BaseServiceImpl 获得标准服务生命周期
// - 使用 NATSManager 统一 NATS 通信
// - 使用 MessageProcessor 和 common/handler 中的 AuthHandler 处理业务逻辑
// - 消除了所有重复的 NATS 和消息处理代码
// - 现在与其他微服务保持一致的架构模式
