package game

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/idle-server/common"
	"github.com/idle-server/common/handler"
	"github.com/idle-server/common/nats"
	"github.com/idle-server/common/service"
	natsio "github.com/nats-io/nats.go"
)

// Service 统一的游戏服务
type Service struct {
	*service.BaseServiceImpl
	natsManager  *nats.Manager
	processor    *handler.MessageProcessor
	players      map[string]*PlayerState
	playersMutex sync.RWMutex
}

// PlayerState 玩家状态
type PlayerState struct {
	PlayerID    string
	ConnectedAt time.Time
	LastActive  time.Time
	GameData    map[string]interface{} // 游戏数据
}

// NewService 创建新的游戏服务
func NewService() service.Service {
	return &Service{
		BaseServiceImpl: service.NewBaseService("Game"),
		players:         make(map[string]*PlayerState),
	}
}

// Start 启动服务
func (s *Service) Start(ctx context.Context) error {
	// 调用基类 Start 方法
	if err := s.BaseServiceImpl.Start(ctx); err != nil {
		return err
	}

	// 初始化 NATS 管理器
	var err error
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

	log.Printf("Game Service started successfully")
	return nil
}

// Stop 停止服务
func (s *Service) Stop(ctx context.Context) error {
	// 调用基类 Stop 方法
	if err := s.BaseServiceImpl.Stop(ctx); err != nil {
		return err
	}

	// 保存所有玩家数据
	s.saveAllPlayerData()

	// 关闭 NATS 管理器
	if s.natsManager != nil {
		s.natsManager.Close()
	}

	log.Printf("Game Service stopped successfully")
	return nil
}

// registerHandlers 注册消息处理器
func (s *Service) registerHandlers() error {
	// 注册玩家连接处理器
	connectHandler := handler.NewPlayerConnectHandler(s.natsManager, s.handlePlayerConnect)
	s.processor.RegisterHandler(connectHandler)

	// 注册玩家断开连接处理器
	disconnectHandler := handler.NewPlayerDisconnectHandler(s.natsManager, s.handlePlayerDisconnect)
	s.processor.RegisterHandler(disconnectHandler)

	// 注册游戏状态处理器
	stateHandler := handler.NewGameStateHandler(s.natsManager, s.handleGetState)
	s.processor.RegisterHandler(stateHandler)

	// 注册游戏动作处理器
	actionHandler := handler.NewGameActionHandler(s.natsManager, s.handleGameAction)
	s.processor.RegisterHandler(actionHandler)

	return nil
}

// registerNATSSubscriptions 注册 NATS 订阅
func (s *Service) registerNATSSubscriptions() error {
	// 使用统一的消息处理器订阅玩家连接主题
	if _, err := s.natsManager.Subscribe(common.GamePlayerConnectSubject, &natsMessageAdapter{
		processor: s.processor,
	}); err != nil {
		return fmt.Errorf("failed to subscribe to player connect subject: %w", err)
	}

	// 使用统一的消息处理器订阅玩家断开连接主题
	if _, err := s.natsManager.Subscribe(common.GamePlayerDisconnectSubject, &natsMessageAdapter{
		processor: s.processor,
	}); err != nil {
		return fmt.Errorf("failed to subscribe to player disconnect subject: %w", err)
	}

	// 使用统一的消息处理器订阅游戏状态主题
	if _, err := s.natsManager.Subscribe(common.GameStateSubject, &natsMessageAdapter{
		processor: s.processor,
	}); err != nil {
		return fmt.Errorf("failed to subscribe to game state subject: %w", err)
	}

	// 使用统一的消息处理器订阅游戏动作主题
	if _, err := s.natsManager.Subscribe(common.GameActionSubject, &natsMessageAdapter{
		processor: s.processor,
	}); err != nil {
		return fmt.Errorf("failed to subscribe to game action subject: %w", err)
	}

	log.Printf("Game NATS subscriptions registered successfully")
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

// 业务逻辑处理方法

// handlePlayerConnect 处理玩家连接
func (s *Service) handlePlayerConnect(playerID string) error {
	log.Printf("Game Service: Player %s connected", playerID)

	s.playersMutex.Lock()
	defer s.playersMutex.Unlock()

	// 创建或更新玩家状态
	playerState := &PlayerState{
		PlayerID:    playerID,
		ConnectedAt: time.Now(),
		LastActive:  time.Now(),
		GameData:    make(map[string]interface{}),
	}

	// 尝试加载玩家数据
	if err := s.loadPlayerData(playerID); err != nil {
		log.Printf("Failed to load player data for %s: %v", playerID, err)
		// 为新玩家初始化默认数据
		playerState.GameData = s.initDefaultGameData()
	}

	s.players[playerID] = playerState

	log.Printf("Player %s connected and initialized successfully", playerID)
	return nil
}

// handlePlayerDisconnect 处理玩家断开连接
func (s *Service) handlePlayerDisconnect(playerID string) error {
	log.Printf("Game Service: Player %s disconnected", playerID)

	s.playersMutex.Lock()
	defer s.playersMutex.Unlock()

	if playerState, exists := s.players[playerID]; exists {
		// 保存玩家数据
		if err := s.savePlayerData(playerID, playerState.GameData); err != nil {
			log.Printf("Failed to save player data for %s: %v", playerID, err)
		}

		// 从内存中移除玩家
		delete(s.players, playerID)
	}

	log.Printf("Player %s disconnected and data saved", playerID)
	return nil
}

// handleGetState 处理获取游戏状态
func (s *Service) handleGetState(playerID string) (interface{}, error) {
	s.playersMutex.RLock()
	defer s.playersMutex.RUnlock()

	playerState, exists := s.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player %s not found", playerID)
	}

	// 更新最后活跃时间
	playerState.LastActive = time.Now()

	// 返回玩家游戏状态
	return map[string]interface{}{
		"player_id":    playerID,
		"connected_at": playerState.ConnectedAt.Unix(),
		"last_active":  playerState.LastActive.Unix(),
		"game_data":    playerState.GameData,
	}, nil
}

// handleGameAction 处理游戏动作
func (s *Service) handleGameAction(playerID, action string, params map[string]interface{}) (interface{}, error) {
	s.playersMutex.Lock()
	defer s.playersMutex.Unlock()

	playerState, exists := s.players[playerID]
	if !exists {
		return nil, fmt.Errorf("player %s not found", playerID)
	}

	// 更新最后活跃时间
	playerState.LastActive = time.Now()

	log.Printf("Processing game action '%s' for player %s", action, playerID)

	// 处理不同的游戏动作
	switch action {
	case "update_level":
		return s.handleUpdateLevel(playerState, params)
	case "add_resource":
		return s.handleAddResource(playerState, params)
	case "save_progress":
		return s.handleSaveProgress(playerState, params)
	default:
		return nil, fmt.Errorf("unknown game action: %s", action)
	}
}

// 游戏动作处理方法

func (s *Service) handleUpdateLevel(playerState *PlayerState, params map[string]interface{}) (interface{}, error) {
	level, ok := params["level"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid level parameter")
	}

	playerState.GameData["level"] = int(level)

	return map[string]interface{}{
		"action": "update_level",
		"result": "success",
		"level":  int(level),
	}, nil
}

func (s *Service) handleAddResource(playerState *PlayerState, params map[string]interface{}) (interface{}, error) {
	resourceType, ok := params["resource_type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid resource_type parameter")
	}

	amount, ok := params["amount"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid amount parameter")
	}

	// 确保资源字典存在
	if playerState.GameData["resources"] == nil {
		playerState.GameData["resources"] = make(map[string]interface{})
	}

	resources := playerState.GameData["resources"].(map[string]interface{})
	currentAmount := float64(0)
	if current, exists := resources[resourceType]; exists {
		if currentFloat, ok := current.(float64); ok {
			currentAmount = currentFloat
		}
	}

	newAmount := currentAmount + amount
	resources[resourceType] = newAmount

	return map[string]interface{}{
		"action":        "add_resource",
		"result":        "success",
		"resource_type": resourceType,
		"new_amount":    newAmount,
	}, nil
}

func (s *Service) handleSaveProgress(playerState *PlayerState, params map[string]interface{}) (interface{}, error) {
	playerID := playerState.PlayerID

	if err := s.savePlayerData(playerID, playerState.GameData); err != nil {
		return nil, fmt.Errorf("failed to save progress: %w", err)
	}

	return map[string]interface{}{
		"action": "save_progress",
		"result": "success",
		"time":   time.Now().Unix(),
	}, nil
}

// 数据管理方法

func (s *Service) initDefaultGameData() map[string]interface{} {
	return map[string]interface{}{
		"level":      1,
		"experience": 0,
		"resources": map[string]interface{}{
			"gold":   100,
			"gems":   10,
			"energy": 100,
		},
		"achievements": []interface{}{},
		"last_save":    time.Now().Unix(),
	}
}

func (s *Service) loadPlayerData(playerID string) error {
	log.Printf("Game: Loading player data for %s", playerID)

	// 从 persist 服务加载玩家数据
	req := map[string]interface{}{
		"type":      "C_LoadPlayer",
		"player_id": playerID,
	}

	var result map[string]interface{}
	err := s.natsManager.RequestWithReply(common.PersistLoadPlayerSubject, req, &result, 5*time.Second)
	if err != nil {
		log.Printf("Game: Failed to get response from Persist: %v", err)
		return err
	}

	log.Printf("Game: Received load response from Persist: %+v", result)

	// 解析响应
	success, ok := result["success"].(bool)
	if !ok || !success {
		message, _ := result["message"].(string)
		log.Printf("Game: Failed to load player data: %s", message)
		return fmt.Errorf("failed to load player data: %s", message)
	}

	// 获取玩家数据并更新到玩家状态
	s.playersMutex.Lock()
	defer s.playersMutex.Unlock()

	if playerState, exists := s.players[playerID]; exists {
		if data, ok := result["data"].(map[string]interface{}); ok {
			if playerData, ok := data["data"].(map[string]interface{}); ok {
				playerState.GameData = playerData
				log.Printf("Game: Successfully loaded player data for %s", playerID)
				return nil
			}
		}
	}

	log.Printf("Game: Invalid response data format for player %s", playerID)
	return fmt.Errorf("invalid response data format")
}

func (s *Service) savePlayerData(playerID string, gameData map[string]interface{}) error {
	log.Printf("Game: Saving player data for %s", playerID)

	// 保存到 persist 服务
	req := map[string]interface{}{
		"type":      "C_SavePlayer",
		"player_id": playerID,
		"data":      gameData,
	}

	err := s.natsManager.Publish(common.PersistSavePlayerSubject, req)
	if err != nil {
		log.Printf("Game: Failed to publish save request: %v", err)
		return err
	}

	log.Printf("Game: Successfully sent save request for player %s", playerID)
	return nil
}

func (s *Service) saveAllPlayerData() {
	s.playersMutex.RLock()
	defer s.playersMutex.RUnlock()

	for playerID, playerState := range s.players {
		if err := s.savePlayerData(playerID, playerState.GameData); err != nil {
			log.Printf("Failed to save data for player %s: %v", playerID, err)
		}
	}

	log.Printf("All player data saved during shutdown")
}

// GetConnectedPlayers 获取连接的玩家数量（用于调试和监控）
func (s *Service) GetConnectedPlayers() int {
	s.playersMutex.RLock()
	defer s.playersMutex.RUnlock()
	return len(s.players)
}

// GetPlayerState 获取玩家状态（用于调试和监控）
func (s *Service) GetPlayerState(playerID string) (*PlayerState, bool) {
	s.playersMutex.RLock()
	defer s.playersMutex.RUnlock()

	state, exists := s.players[playerID]
	return state, exists
}
