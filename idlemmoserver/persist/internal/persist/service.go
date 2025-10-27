package persist

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/idle-server/common"
	"github.com/idle-server/common/database"
	"github.com/idle-server/common/handler"
	"github.com/idle-server/common/nats"
	"github.com/idle-server/common/service"
	natsio "github.com/nats-io/nats.go"
)

// Service 统一的持久化服务
type Service struct {
	*service.BaseServiceImpl
	natsManager       *nats.Manager
	processor         *handler.MessageProcessor
	gormDB            *database.GORM
	redis             *database.Redis
	userRepo          *database.GORMUserRepository
	playerRepo        *database.GORMPlayerRepository
	healthCheckCtx    context.Context
	healthCheckCancel context.CancelFunc
}

// NewService 创建新的持久化服务
func NewService() service.Service {
	return &Service{
		BaseServiceImpl: service.NewBaseService("Persist"),
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	// 调用基类 Start 方法
	if err := s.BaseServiceImpl.Start(ctx); err != nil {
		return err
	}

	log.Println("Initializing Persist Service (MySQL + Redis + GORM)...")

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
	s.playerRepo = database.NewGORMPlayerRepository(gormDB.GetDB(), redis)

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

	// 启动健康检查
	s.healthCheckCtx, s.healthCheckCancel = context.WithCancel(ctx)
	go s.startHealthCheck(s.healthCheckCtx, 30*time.Second)

	log.Printf("Persist Service started successfully with MySQL, Redis and GORM")
	return nil
}

// Stop 停止服务
func (s *Service) Stop(ctx context.Context) error {
	// 调用基类 Stop 方法
	if err := s.BaseServiceImpl.Stop(ctx); err != nil {
		return err
	}

	log.Println("Stopping Persist Service...")

	// 停止健康检查
	if s.healthCheckCancel != nil {
		s.healthCheckCancel()
	}

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

	log.Println("Persist Service stopped successfully")
	return nil
}

// registerHandlers 注册消息处理器
func (s *Service) registerHandlers() error {
	// 注册用户保存处理器
	saveUserHandler := handler.NewSaveUserHandler(s.natsManager, s.saveUserData)
	s.processor.RegisterHandler(saveUserHandler)

	// 注册用户加载处理器
	loadUserHandler := handler.NewLoadUserHandler(s.natsManager, s.loadUserData)
	s.processor.RegisterHandler(loadUserHandler)

	// 注册玩家保存处理器
	savePlayerHandler := handler.NewSavePlayerHandler(s.natsManager, s.savePlayerData)
	s.processor.RegisterHandler(savePlayerHandler)

	// 注册玩家加载处理器
	loadPlayerHandler := handler.NewLoadPlayerHandler(s.natsManager, s.loadPlayerData)
	s.processor.RegisterHandler(loadPlayerHandler)

	// 注册用户删除处理器
	deleteUserHandler := handler.NewDeleteUserHandler(s.natsManager, s.deleteUserData)
	s.processor.RegisterHandler(deleteUserHandler)

	return nil
}

// registerNATSSubscriptions 注册 NATS 订阅
func (s *Service) registerNATSSubscriptions() error {
	// 创建消息处理器适配器
	adapter := &natsMessageAdapter{processor: s.processor}

	// 用户相关订阅
	subjects := []string{
		common.PersistSaveUserSubject,
		common.PersistLoadUserSubject,
		common.PersistUserExistsSubject,
		common.PersistSaveSubject,
		common.PersistLoadSubject,
		"persist.create_user",
		"persist.authenticate_user",
		"persist.player_exists",
		"persist.update_player_status",
	}

	for _, subject := range subjects {
		if _, err := s.natsManager.Subscribe(subject, adapter); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}
	}

	log.Println("NATS handlers registered successfully")
	return nil
}

// natsMessageAdapter 消息处理器适配器
type natsMessageAdapter struct {
	processor *handler.MessageProcessor
}

// Handle 实现 nats.MessageHandler 接口
func (a *natsMessageAdapter) Handle(msg *natsio.Msg) error {
	return a.processor.ProcessMessage(msg)
}

// ============ 业务逻辑方法 ============

// saveUserData 保存用户数据业务逻辑
func (s *Service) saveUserData(userID string, data interface{}) error {
	log.Printf("saveUserData called with userID: %s, data type: %T", userID, data)

	var userData *common.UserData
	var err error

	// 尝试将数据转换为 UserData 结构体
	switch v := data.(type) {
	case *common.UserData:
		userData = v
		log.Printf("Received data as *common.UserData")
	case map[string]interface{}:
		log.Printf("Received data as map[string]interface{}, converting to UserData")
		userData, err = s.mapToUserData(v)
		if err != nil {
			log.Printf("Failed to convert map to UserData: %v", err)
			return fmt.Errorf("failed to convert user data: %w", err)
		}
	default:
		log.Printf("Failed to convert data to *common.UserData, actual type: %T, data: %+v", data, data)
		return fmt.Errorf("invalid user data type, got %T", data)
	}

	log.Printf("Saving user data for: %s (PlayerID: %s)", userData.Username, userData.PlayerID)
	log.Printf("UserData details: Username=%s, Password=%s, PlayerID=%s", userData.Username, userData.Password, userData.PlayerID)

	// 检查必要的字段
	if userData.Username == "" {
		return fmt.Errorf("missing username in user data")
	}
	if userData.Password == "" {
		return fmt.Errorf("missing password in user data")
	}

	// 使用 GORM 仓库保存用户 - CreateUser 会自动生成 PlayerID
	savedUserData, err := s.userRepo.CreateUser(s.healthCheckCtx, userData.Username, userData.Password)
	if err != nil {
		log.Printf("Failed to save user %s: %v", userData.Username, err)
		return fmt.Errorf("failed to save user data: %w", err)
	}

	log.Printf("User data saved successfully for: %s (Generated PlayerID: %s)", savedUserData.Username, savedUserData.PlayerID)
	return nil
}

// mapToUserData 将 map[string]interface{} 转换为 UserData 结构体
func (s *Service) mapToUserData(data map[string]interface{}) (*common.UserData, error) {
	userData := &common.UserData{}

	// 提取 username
	if username, ok := data["username"].(string); ok {
		userData.Username = username
	} else {
		return nil, fmt.Errorf("missing or invalid username")
	}

	// 提取 password
	if password, ok := data["password"].(string); ok {
		userData.Password = password
	} else {
		return nil, fmt.Errorf("missing or invalid password")
	}

	// 提取 player_id (可选)
	if playerID, ok := data["player_id"].(string); ok {
		userData.PlayerID = playerID
	}

	// 其他字段为可选，使用默认值

	return userData, nil
}

// loadUserData 加载用户数据业务逻辑
func (s *Service) loadUserData(userID string) (interface{}, error) {
	log.Printf("Loading user data for: %s", userID)

	userData, err := s.userRepo.GetUserByUsername(s.healthCheckCtx, userID)
	if err != nil {
		log.Printf("Failed to load user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to load user data: %w", err)
	}

	log.Printf("User data loaded successfully for: %s", userID)
	return userData, nil
}

// savePlayerData 保存玩家数据业务逻辑
func (s *Service) savePlayerData(playerID string, data interface{}) error {
	playerData, ok := data.(*common.PlayerData)
	if !ok {
		return fmt.Errorf("invalid player data type")
	}

	log.Printf("Saving player data for: %s", playerID)

	err := s.playerRepo.SavePlayerData(s.healthCheckCtx, playerData)
	if err != nil {
		log.Printf("Failed to save player %s: %v", playerID, err)
		return fmt.Errorf("failed to save player data: %w", err)
	}

	log.Printf("Player data saved successfully for: %s", playerID)
	return nil
}

// loadPlayerData 加载玩家数据业务逻辑
func (s *Service) loadPlayerData(playerID string) (interface{}, error) {
	log.Printf("Loading player data for: %s", playerID)

	playerData, err := s.playerRepo.LoadPlayerData(s.healthCheckCtx, playerID)
	if err != nil {
		log.Printf("Failed to load player %s: %v", playerID, err)
		return nil, fmt.Errorf("failed to load player data: %w", err)
	}

	log.Printf("Player data loaded successfully for: %s", playerID)
	return playerData, nil
}

// deleteUserData 删除用户数据业务逻辑
func (s *Service) deleteUserData(userID string) error {
	log.Printf("Deleting user data for: %s", userID)

	if err := s.userRepo.DeleteUser(s.healthCheckCtx, userID); err != nil {
		log.Printf("Failed to delete user %s: %v", userID, err)
		return fmt.Errorf("failed to delete user data: %w", err)
	}

	log.Printf("User data deleted successfully for: %s", userID)
	return nil
}

// ============ 健康检查和辅助方法 ============

// startHealthCheck 启动健康检查
func (s *Service) startHealthCheck(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Health check stopped")
			return
		case <-ticker.C:
			s.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (s *Service) performHealthCheck() {
	// 检查GORM数据库连接
	if s.gormDB != nil && !s.gormDB.IsHealthy() {
		log.Printf("GORM database health check failed")
	}

	// 检查Redis连接
	if s.redis != nil {
		if err := s.redis.Ping(context.Background()); err != nil {
			log.Printf("Redis health check failed: %v", err)
		}
	}

	// 检查NATS连接
	if s.natsManager != nil && !s.natsManager.IsConnected() {
		log.Printf("NATS connection health check failed")
	}
}

// Persist 服务重构完成
// 使用统一的架构：
// - 继承 BaseServiceImpl 获得标准服务生命周期
// - 使用 NATSManager 统一 NATS 通信
// - 使用 MessageProcessor 和具体 Handler 处理业务逻辑
// - 分离了数据操作（GORM Repository）和业务逻辑
// - 消除了所有重复的 NATS 和消息处理代码
