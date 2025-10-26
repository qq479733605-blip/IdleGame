package domain

import (
	"math/rand"
	"time"

	"github.com/idle-server/common"
)

// 原有类型定义已移至common模块，此处使用common包中的类型

// Tick 为Sequence提供时间计算功能
func TickSequence(s *common.Sequence, cfg *common.SequenceConfig, bonus common.EquipmentBonus) common.TickResult {
	// 复制掉落配置，避免修改原数据
	drops := make([]common.Item, len(cfg.Drops))
	copy(drops, cfg.Drops)

	gainMultiplier := 1.0 + bonus.GainMultiplier
	rareChance := cfg.RareChance + bonus.RareChanceBonus
	expRate := cfg.ExpRate * (1.0 + bonus.ExpMultiplier)

	if s.SubProject != nil {
		if len(s.SubProject.ExtraDrops) > 0 {
			drops = append(drops, s.SubProject.ExtraDrops...)
		}
		if s.SubProject.GainMultiplier > 0 {
			gainMultiplier *= s.SubProject.GainMultiplier
		}
		rareChance += s.SubProject.RareChanceBonus
		if s.SubProject.ExpMultiplier > 0 {
			expRate *= s.SubProject.ExpMultiplier
		}
	}

	if rareChance < 0 {
		rareChance = 0
	} else if rareChance > 1 {
		rareChance = 1
	}

	// 基础收益 + 成长
	gains := cfg.BaseGain + int64(float64(s.Level)*cfg.GrowthFactor)
	gains = int64(float64(gains) * gainMultiplier)
	if gains < 0 {
		gains = 0
	}

	// 掉落
	items := []common.Item{}
	for _, it := range drops {
		if rand.Float64() < it.DropChance {
			items = append(items, it)
		}
	}

	// 装备掉落
	for _, equip := range cfg.EquipmentDrops {
		if s.Level >= equip.MinLevel && rand.Float64() < equip.DropChance {
			items = append(items, common.Item{
				ID:          equip.ID,
				Name:        equip.Name,
				DropChance:  equip.DropChance,
				Value:       50,   // 装备基础价值
				IsEquipment: true, // 标记为装备
			})
		}
	}

	// 奇遇
	var rare *common.RareEvent
	if rand.Float64() < rareChance && len(cfg.RareEvents) > 0 {
		r := cfg.RareEvents[rand.Intn(len(cfg.RareEvents))]
		rare = &r
		gains = int64(float64(gains) * rare.MultGain)
	}

	// 成长：经验与升级（表驱动）
	expGain := int64(float64(gains) * expRate)
	s.Exp += expGain
	leveled := false
	if s.Exp >= cfg.LevelUpExp {
		s.Level++
		s.Exp = 0
		leveled = true
	}

	s.LastTick = time.Now()
	subID := ""
	if s.SubProject != nil {
		subID = s.SubProject.ID
	}
	return common.TickResult{
		Gains:        gains,
		Items:        items,
		RareEvt:      rare,
		Level:        s.Level,
		CurExp:       s.Exp,
		Leveled:      leveled,
		SubProjectID: subID,
	}
}

// SetSubProject 设置序列的子项目
func SetSubProject(s *common.Sequence, sp *common.SequenceSubProject) {
	s.SubProject = sp
}

// EffectiveInterval 计算有效间隔时间
func EffectiveInterval(cfg *common.SequenceConfig, sp *common.SequenceSubProject) time.Duration {
	base := float64(cfg.TickInterval)
	if base <= 0 {
		base = 1
	}
	if sp != nil && sp.IntervalModifier > 0 {
		base = base * sp.IntervalModifier
	}
	if base < 0.5 {
		base = 0.5
	}
	return time.Duration(base * float64(time.Second))
}
