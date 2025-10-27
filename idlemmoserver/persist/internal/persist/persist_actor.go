package persist

import (
	"encoding/json"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// PersistActor 持久化Actor - 基于原有实现适配
type PersistActor struct {
	repo   *JSONRepository
	online map[string]*actor.PID // 已注册玩家
	ticker *time.Ticker
	nc     *nats.Conn
}

// NewPersistActor 创建持久化Actor
func NewPersistActor(repo *JSONRepository, nc *nats.Conn) actor.Actor {
	return &PersistActor{
		repo:   repo,
		online: make(map[string]*actor.PID),
		nc:     nc,
	}
}

// Receive 处理消息
func (p *PersistActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		p.handleStarted(ctx)
	case *common.MsgSavePlayer:
		p.handleSavePlayer(ctx, msg)
	case *common.MsgLoadPlayer:
		p.handleLoadPlayer(ctx, msg)
	case *common.MsgRegisterPlayer:
		p.handleRegisterPlayer(ctx, msg)
	case *common.MsgUnregisterPlayer:
		p.handleUnregisterPlayer(ctx, msg)
	case *common.MsgSaveUser:
		p.handleSaveUser(ctx, msg)
	case *common.MsgLoadUser:
		p.handleLoadUser(ctx, msg)
	case *common.MsgUserExists:
		p.handleUserExists(ctx, msg)
	case *actor.Stopping:
		p.handleStopping(ctx)
	default:
		log.Printf("PersistActor: unknown message type %T", msg)
	}
}

// handleStarted 处理Actor启动
func (p *PersistActor) handleStarted(ctx actor.Context) {
	log.Println("PersistActor started")

	// 每 60 秒检查一次离线超时
	p.ticker = time.NewTicker(60 * time.Second)
	go func() {
		for range p.ticker.C {
			p.checkOfflineTimeouts(ctx)
		}
	}()

	// 注册NATS处理器
	p.registerNATSHandlers(ctx)
}

// handleSavePlayer 处理保存玩家数据
func (p *PersistActor) handleSavePlayer(ctx actor.Context, msg *common.MsgSavePlayer) {
	err := p.repo.Save(msg.PlayerID, msg.PlayerData)
	if err != nil {
		log.Printf("Failed to save player %s: %v", msg.PlayerID, err)
	} else {
		log.Printf("Player %s saved successfully", msg.PlayerID)
	}
}

// handleLoadPlayer 处理加载玩家数据
func (p *PersistActor) handleLoadPlayer(ctx actor.Context, msg *common.MsgLoadPlayer) {
	data, err := p.repo.Load(msg.PlayerID)

	// 发送回复
	reply := &common.MsgLoadResult{
		Data: data,
		Err:  err,
	}

	// 通过Actor系统发送回复
	if msg.ReplyTo != nil {
		ctx.Send(msg.ReplyTo, reply)
	}
}

// handleRegisterPlayer 处理玩家注册
func (p *PersistActor) handleRegisterPlayer(ctx actor.Context, msg *common.MsgRegisterPlayer) {
	p.online[msg.PlayerID] = msg.PID
	log.Printf("Player registered for persistence: %s", msg.PlayerID)
}

// handleUnregisterPlayer 处理玩家注销
func (p *PersistActor) handleUnregisterPlayer(ctx actor.Context, msg *common.MsgUnregisterPlayer) {
	delete(p.online, msg.PlayerID)
	log.Printf("Player unregistered from persistence: %s", msg.PlayerID)
}

// handleStopping 处理Actor停止
func (p *PersistActor) handleStopping(ctx actor.Context) {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	log.Println("PersistActor stopping")
}

// checkOfflineTimeouts 检查离线超时
func (p *PersistActor) checkOfflineTimeouts(ctx actor.Context) {
	// TODO: 实现离线超时检查逻辑
	log.Println("Checking offline timeouts")
}

// registerNATSHandlers 注册NATS处理器
func (p *PersistActor) registerNATSHandlers(ctx actor.Context) {
	// 注册保存处理器
	_, err := p.nc.Subscribe(common.PersistSaveSubject, func(msg *nats.Msg) {
		p.handleNATSSave(msg)
	})
	if err != nil {
		log.Printf("Failed to register save handler: %v", err)
		return
	}

	// 注册加载处理器
	_, err = p.nc.Subscribe(common.PersistLoadSubject, func(msg *nats.Msg) {
		p.handleNATSLoad(msg)
	})
	if err != nil {
		log.Printf("Failed to register load handler: %v", err)
		return
	}

	// 注册用户保存处理器
	_, err = p.nc.Subscribe(common.PersistSaveUserSubject, func(msg *nats.Msg) {
		p.handleNATSSaveUser(msg)
	})
	if err != nil {
		log.Printf("Failed to register save user handler: %v", err)
		return
	}

	// 注册用户加载处理器
	_, err = p.nc.Subscribe(common.PersistLoadUserSubject, func(msg *nats.Msg) {
		p.handleNATSLoadUser(msg)
	})
	if err != nil {
		log.Printf("Failed to register load user handler: %v", err)
		return
	}

	// 注册用户存在检查处理器
	_, err = p.nc.Subscribe(common.PersistUserExistsSubject, func(msg *nats.Msg) {
		p.handleNATSUserExists(msg)
	})
	if err != nil {
		log.Printf("Failed to register user exists handler: %v", err)
		return
	}

	log.Println("Persist NATS handlers registered")
}

// handleNATSSave 处理NATS保存请求
func (p *PersistActor) handleNATSSave(msg *nats.Msg) {
	var saveMsg common.MsgSavePlayer
	if err := json.Unmarshal(msg.Data, &saveMsg); err != nil {
		log.Printf("Failed to unmarshal save message: %v", err)
		return
	}

	// 处理保存
	err := p.repo.Save(saveMsg.PlayerID, saveMsg.PlayerData)
	if err != nil {
		log.Printf("Failed to save via NATS: %v", err)
	}
}

// handleNATSLoad 处理NATS加载请求
func (p *PersistActor) handleNATSLoad(msg *nats.Msg) {
	var loadMsg common.MsgLoadPlayer
	if err := json.Unmarshal(msg.Data, &loadMsg); err != nil {
		log.Printf("Failed to unmarshal load message: %v", err)
		return
	}

	// 处理加载
	data, err := p.repo.Load(loadMsg.PlayerID)

	// 发送回复
	reply := common.MsgLoadResult{
		Data: data,
		Err:  err,
	}

	replyData, _ := json.Marshal(reply)
	msg.Respond(replyData)
}

// ========== 用户数据NATS处理方法 ==========

// handleNATSSaveUser 处理NATS用户保存请求
func (p *PersistActor) handleNATSSaveUser(msg *nats.Msg) {
	log.Printf("🔥 Persist service received save user request via NATS!")
	log.Printf("🔥 Request data: %s", string(msg.Data))

	var saveMsg common.MsgSaveUser
	if err := json.Unmarshal(msg.Data, &saveMsg); err != nil {
		log.Printf("Failed to unmarshal save user message: %v", err)
		return
	}

	log.Printf("🔥 Successfully unmarshaled save user request for user: %s", saveMsg.UserData.Username)

	// 处理保存
	err := p.repo.SaveUser(saveMsg.UserData)
	if err != nil {
		log.Printf("Failed to save user via NATS: %v", err)
		// 即使失败也要发送回复
		msg.Respond([]byte(`{"success": false, "message": "保存失败"}`))
	} else {
		log.Printf("🔥 Successfully saved user: %s", saveMsg.UserData.Username)
		// 发送成功回复
		msg.Respond([]byte(`{"success": true, "message": "保存成功"}`))
	}
}

// handleNATSLoadUser 处理NATS用户加载请求
func (p *PersistActor) handleNATSLoadUser(msg *nats.Msg) {
	var loadMsg common.MsgLoadUser
	if err := json.Unmarshal(msg.Data, &loadMsg); err != nil {
		log.Printf("Failed to unmarshal load user message: %v", err)
		return
	}

	// 处理加载
	userData, err := p.repo.LoadUser(loadMsg.Username)

	// 发送回复
	reply := common.MsgLoadUserResult{
		UserData: userData,
		Err:      err,
	}

	replyData, _ := json.Marshal(reply)
	msg.Respond(replyData)
}

// handleNATSUserExists 处理NATS用户存在检查请求
func (p *PersistActor) handleNATSUserExists(msg *nats.Msg) {
	log.Printf("🔥 Persist service received user exists request via NATS!")
	log.Printf("🔥 Request data: %s", string(msg.Data))

	var existsMsg common.MsgUserExists
	if err := json.Unmarshal(msg.Data, &existsMsg); err != nil {
		log.Printf("Failed to unmarshal user exists message: %v", err)
		return
	}

	log.Printf("🔥 Successfully unmarshaled user exists request for user: %s", existsMsg.Username)

	// 处理检查
	exists := p.repo.UserExists(existsMsg.Username)

	// 发送回复
	reply := common.MsgUserExistsResult{
		Exists: exists,
	}

	replyData, _ := json.Marshal(reply)
	msg.Respond(replyData)
}

// ========== 用户数据处理方法 ==========

// handleSaveUser 处理保存用户数据
func (p *PersistActor) handleSaveUser(ctx actor.Context, msg *common.MsgSaveUser) {
	err := p.repo.SaveUser(msg.UserData)
	if err != nil {
		log.Printf("Failed to save user %s: %v", msg.UserData.Username, err)
	} else {
		log.Printf("User %s saved successfully", msg.UserData.Username)
	}
}

// handleLoadUser 处理加载用户数据
func (p *PersistActor) handleLoadUser(ctx actor.Context, msg *common.MsgLoadUser) {
	userData, err := p.repo.LoadUser(msg.Username)

	// 发送回复
	reply := &common.MsgLoadUserResult{
		UserData: userData,
		Err:      err,
	}

	// 通过Actor系统发送回复
	if msg.ReplyTo != nil {
		ctx.Send(msg.ReplyTo, reply)
	}
}

// handleUserExists 处理检查用户是否存在
func (p *PersistActor) handleUserExists(ctx actor.Context, msg *common.MsgUserExists) {
	exists := p.repo.UserExists(msg.Username)

	// 发送回复
	reply := &common.MsgUserExistsResult{
		Exists: exists,
	}

	// 通过Actor系统发送回复
	if msg.ReplyTo != nil {
		ctx.Send(msg.ReplyTo, reply)
	}
}
