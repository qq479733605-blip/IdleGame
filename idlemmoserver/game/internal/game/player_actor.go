package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// PlayerActor 玩家Actor - 基于原有实现适配
type PlayerActor struct {
	playerID       string
	system         *actor.ActorSystem
	currentSeq     *actor.PID
	currentSeqID   string
	playerData     *common.PlayerData
	equipmentBonus common.EquipmentBonus
	nc             *nats.Conn
	lastSaveTime   time.Time
	offlineLimit   time.Duration
}

// NewPlayerActor 创建玩家Actor
func NewPlayerActor(playerID string, system *actor.ActorSystem, nc *nats.Conn) actor.Actor {
	return &PlayerActor{
		playerID:       playerID,
		system:         system,
		playerData:     createNewPlayerData(playerID),
		equipmentBonus: common.EquipmentBonus{},
		nc:             nc,
		lastSaveTime:   time.Now(),
		offlineLimit:   24 * time.Hour, // 默认24小时离线限制
	}
}

// Receive 处理消息
func (p *PlayerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		p.handleStarted(ctx)
	case *common.MsgStartSequence:
		p.handleStartSequence(ctx, msg)
	case *common.MsgStopSequence:
		p.handleStopSequence(ctx, msg)
	case *common.MsgSequenceResult:
		p.handleSequenceResult(ctx, msg)
	case *common.MsgLoadResult:
		p.handleLoadResult(ctx, msg)
	case *common.MsgUpdateEquipmentBonus:
		p.handleUpdateEquipmentBonus(ctx, msg)
	default:
		log.Printf("PlayerActor(%s): unknown message type %T", p.playerID, msg)
	}
}

// handleStarted 处理Actor启动
func (p *PlayerActor) handleStarted(ctx actor.Context) {
	log.Printf("PlayerActor started for player: %s", p.playerID)

	// 加载玩家数据
	p.loadPlayerData(ctx)

	// 计算装备加成
	p.calculateEquipmentBonus()
}

// handleStartSequence 处理开始序列
func (p *PlayerActor) handleStartSequence(ctx actor.Context, msg *common.MsgStartSequence) {
	// 如果已有序列在运行，先停止
	if p.currentSeq != nil {
		p.stopCurrentSequence(ctx)
	}

	// 创建序列Actor
	seqProps := actor.PropsFromProducer(func() actor.Actor {
		return NewSequenceActor(p.playerID)
	})
	p.currentSeq = ctx.Spawn(seqProps)
	p.currentSeqID = msg.SeqID

	// 发送开始消息
	ctx.Send(p.currentSeq, msg)

	log.Printf("Started sequence %s for player %s", msg.SeqID, p.playerID)
}

// handleStopSequence 处理停止序列
func (p *PlayerActor) handleStopSequence(ctx actor.Context, msg *common.MsgStopSequence) {
	p.stopCurrentSequence(ctx)
}

// handleSequenceResult 处理序列结果
func (p *PlayerActor) handleSequenceResult(ctx actor.Context, msg *common.MsgSequenceResult) {
	// 更新玩家数据
	p.updatePlayerData(msg)

	// 定期保存数据
	if time.Since(p.lastSaveTime) > 5*time.Minute {
		p.savePlayerData()
	}

	// 广播结果给客户端
	p.broadcastToClient(msg)
}

// handleLoadResult 处理加载结果
func (p *PlayerActor) handleLoadResult(ctx actor.Context, msg *common.MsgLoadResult) {
	if msg.Err != nil {
		log.Printf("Failed to load player data for %s: %v", p.playerID, msg.Err)
		// 使用默认数据
		p.playerData = createNewPlayerData(p.playerID)
	} else {
		p.playerData = msg.Data
	}

	log.Printf("Player data loaded for %s", p.playerID)
	p.calculateEquipmentBonus()
}

// handleUpdateEquipmentBonus 处理装备加成更新
func (p *PlayerActor) handleUpdateEquipmentBonus(ctx actor.Context, msg *common.MsgUpdateEquipmentBonus) {
	p.equipmentBonus = msg.Bonus

	// 如果有序列在运行，更新加成
	if p.currentSeq != nil {
		ctx.Send(p.currentSeq, msg)
	}
}

// loadPlayerData 加载玩家数据
func (p *PlayerActor) loadPlayerData(ctx actor.Context) {
	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &LoadDataReplyActor{playerActor: ctx.Self()}
	})
	replyPID := ctx.Spawn(replyProps)

	// 发送加载请求到持久化服务
	loadMsg := common.MsgLoadPlayer{
		PlayerID: p.playerID,
		ReplyTo:  replyPID,
	}

	// 通过NATS发送请求
	p.sendNATSRequest(common.PersistLoadSubject, &loadMsg)
}

// savePlayerData 保存玩家数据
func (p *PlayerActor) savePlayerData() {
	saveMsg := common.MsgSavePlayer{
		PlayerID:   p.playerID,
		PlayerData: p.playerData,
	}

	p.sendNATSPublish(common.PersistSaveSubject, &saveMsg)
	p.lastSaveTime = time.Now()
}

// updatePlayerData 更新玩家数据
func (p *PlayerActor) updatePlayerData(msg *common.MsgSequenceResult) {
	// 更新序列等级
	if p.playerData.SeqLevels == nil {
		p.playerData.SeqLevels = make(map[string]int)
	}
	p.playerData.SeqLevels[msg.SeqID] = msg.Result.Level

	// 添加物品到背包
	if p.playerData.Inventory != nil {
		for _, item := range msg.Result.Items {
			p.inventoryAddItem(item)
		}
	}

	// 更新经验
	p.playerData.Exp += msg.Result.Gains
}

// inventoryAddItem 添加物品到背包
func (p *PlayerActor) inventoryAddItem(item common.Item) {
	// TODO: 实现背包添加逻辑
	log.Printf("Added item %s to player %s inventory", item.Name, p.playerID)
}

// calculateEquipmentBonus 计算装备加成
func (p *PlayerActor) calculateEquipmentBonus() {
	bonus := common.EquipmentBonus{}

	if p.playerData.Equipment != nil {
		for _, equipState := range p.playerData.Equipment {
			if equipState.IsWorn {
				// 累加装备属性
				if gainMult, exists := equipState.Equipment.Attributes["gain_multiplier"]; exists {
					bonus.GainMultiplier += gainMult
				}
				if rareChance, exists := equipState.Equipment.Attributes["rare_chance_bonus"]; exists {
					bonus.RareChanceBonus += rareChance
				}
				if expMult, exists := equipState.Equipment.Attributes["exp_multiplier"]; exists {
					bonus.ExpMultiplier += expMult
				}
			}
		}
	}

	p.equipmentBonus = bonus
}

// stopCurrentSequence 停止当前序列
func (p *PlayerActor) stopCurrentSequence(ctx actor.Context) {
	if p.currentSeq != nil {
		ctx.Send(p.currentSeq, &common.MsgSequenceStop{})
		ctx.Stop(p.currentSeq)
		p.currentSeq = nil
		p.currentSeqID = ""
	}
}

// broadcastToClient 广播消息给客户端
func (p *PlayerActor) broadcastToClient(msg *common.MsgSequenceResult) {
	// 创建服务端消息
	serverMsg := common.S_SeqResult{
		Type:       common.ServerMsgTypeSeqResult,
		SeqID:      msg.SeqID,
		Result:     msg.Result,
		StopReason: msg.StopReason,
	}

	data, _ := json.Marshal(serverMsg)

	// 发送到网关进行广播
	broadcastMsg := common.MsgToClient{
		PlayerID: p.playerID,
		Data:     data,
	}

	p.sendNATSPublish(common.GatewayBroadcastSubject, &broadcastMsg)
}

// sendNATSRequest 发送NATS请求
func (p *PlayerActor) sendNATSRequest(subject string, msg interface{}) {
	data, _ := json.Marshal(msg)
	// 使用同步请求
	resp, err := p.nc.Request(subject, data, 5*time.Second)
	if err != nil {
		log.Printf("NATS request failed: %v", err)
		return
	}
	// 处理响应
	if resp != nil {
		log.Printf("Received NATS response: %s", string(resp.Data))
	}
}

// sendNATSPublish 发送NATS消息
func (p *PlayerActor) sendNATSPublish(subject string, msg interface{}) {
	data, _ := json.Marshal(msg)
	p.nc.Publish(subject, data)
}

// createNewPlayerData 创建新玩家数据
func createNewPlayerData(playerID string) *common.PlayerData {
	return &common.PlayerData{
		PlayerID:          playerID,
		SeqLevels:         make(map[string]int),
		Inventory:         &common.Inventory{Items: make(map[string]*common.ItemStack), MaxSize: 30},
		Exp:               0,
		Equipment:         make(map[string]common.EquipmentState),
		OfflineLimitHours: 24,
		LastSaveTime:      time.Now(),
		CreatedAt:         time.Now(),
	}
}

// LoadDataReplyActor 加载数据回复Actor
type LoadDataReplyActor struct {
	playerActor *actor.PID
}

// Receive 处理回复
func (a *LoadDataReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgLoadResult:
		ctx.Send(a.playerActor, msg)
		ctx.Stop(ctx.Self())
	}
}
