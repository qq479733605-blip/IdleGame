package game

import (
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/idle-server/game/internal/game/domain"
)

// SequenceActor 序列Actor - 基于原有实现适配
type SequenceActor struct {
	playerID       string
	seq            *common.Sequence
	cfg            *domain.SequenceConfig
	parent         *actor.PID
	tickTimer      *time.Timer
	equipmentBonus common.EquipmentBonus
}

// NewSequenceActor 创建序列Actor
func NewSequenceActor(playerID string) actor.Actor {
	return &SequenceActor{
		playerID: playerID,
	}
}

// Receive 处理消息
func (s *SequenceActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgStartSequence:
		s.handleStartSequence(ctx, msg)
	case *common.MsgStopSequence:
		s.handleStopSequence(ctx, msg)
	case *common.MsgSequenceTick:
		s.handleSequenceTick(ctx)
	case *common.MsgUpdateEquipmentBonus:
		s.handleUpdateEquipmentBonus(ctx, msg)
	case *actor.Started:
		log.Printf("SequenceActor started for player: %s", s.playerID)
	case *actor.Stopping:
		if s.tickTimer != nil {
			s.tickTimer.Stop()
		}
		log.Printf("SequenceActor stopping for player: %s", s.playerID)
	}
}

// handleStartSequence 处理开始序列
func (s *SequenceActor) handleStartSequence(ctx actor.Context, msg *common.MsgStartSequence) {
	// 获取序列配置
	cfg, ok := domain.GetSequenceConfig(msg.SeqID)
	if !ok {
		log.Printf("Sequence config not found: %s", msg.SeqID)
		return
	}

	// 创建序列
	s.seq = &common.Sequence{
		ID:        msg.SeqID,
		Level:     1, // TODO: 从玩家数据获取
		Exp:       0,
		StartTime: time.Now(),
		LastTick:  time.Now(),
	}

	s.cfg = cfg

	// 启动定时器
	s.startTickTimer(ctx)

	log.Printf("Sequence started for player %s: %s", s.playerID, msg.SeqID)
}

// handleStopSequence 处理停止序列
func (s *SequenceActor) handleStopSequence(ctx actor.Context, msg *common.MsgStopSequence) {
	s.stopSequence("stopped")
}

// handleSequenceTick 处理序列Tick
func (s *SequenceActor) handleSequenceTick(ctx actor.Context) {
	if s.seq == nil || s.cfg == nil {
		return
	}

	// 执行Tick逻辑
	result := domain.TickSequence(s.seq, &s.cfg.SequenceConfig, s.equipmentBonus)

	// 发送结果给父Actor
	resultMsg := &common.MsgSequenceResult{
		PlayerID:   s.playerID,
		SeqID:      s.seq.ID,
		Result:     result,
		StopReason: "", // 继续运行
	}

	// 检查是否达到目标 (暂时移除目标检查，因为MsgSequenceTick没有Target字段)
	// TODO: 如果需要目标检查，应该通过其他方式传递目标信息

	// 检查是否升级
	if result.Leveled {
		log.Printf("Player %s sequence %s leveled to %d", s.playerID, s.seq.ID, result.Level)
	}

	// 继续下一个Tick
	s.startTickTimer(ctx)

	// 发送结果
	ctx.Send(ctx.Parent(), resultMsg)
}

// handleUpdateEquipmentBonus 处理装备加成更新
func (s *SequenceActor) handleUpdateEquipmentBonus(ctx actor.Context, msg *common.MsgUpdateEquipmentBonus) {
	s.equipmentBonus = msg.Bonus
}

// startTickTimer 启动Tick定时器
func (s *SequenceActor) startTickTimer(ctx actor.Context) {
	interval := domain.EffectiveInterval(&s.cfg.SequenceConfig, s.seq.SubProject)

	s.tickTimer = time.AfterFunc(interval, func() {
		ctx.Send(ctx.Self(), &common.MsgSequenceTick{})
	})
}

// stopSequence 停止序列
func (s *SequenceActor) stopSequence(reason string) {
	if s.tickTimer != nil {
		s.tickTimer.Stop()
		s.tickTimer = nil
	}
	log.Printf("Sequence stopped for player %s: %s (reason: %s)", s.playerID, s.seq.ID, reason)
}
