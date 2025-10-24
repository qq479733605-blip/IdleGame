package domain

import (
	"math/rand"
	"time"
)

type Sequence struct {
	ID        string
	Level     int
	Exp       int64
	StartTime time.Time
	LastTick  time.Time
}

type TickResult struct {
	Gains   int64
	Items   []Item
	RareEvt *RareEvent
	Level   int   // 当前等级（Tick 后）
	CurExp  int64 // 当前经验（Tick 后）
	Leveled bool  // 是否在本次升级
}

func (s *Sequence) Tick(cfg *SequenceConfig) TickResult {
	// 基础收益 + 成长
	gains := cfg.BaseGain + int64(float64(s.Level)*cfg.GrowthFactor)

	// 掉落
	items := []Item{}
	for _, it := range cfg.Drops {
		if rand.Float64() < it.DropChance {
			items = append(items, it)
		}
	}

	// 奇遇
	var rare *RareEvent
	if rand.Float64() < cfg.RareChance && len(cfg.RareEvents) > 0 {
		r := cfg.RareEvents[rand.Intn(len(cfg.RareEvents))]
		rare = &r
		gains = int64(float64(gains) * rare.MultGain)
	}

	// 成长：经验与升级（表驱动）
	expGain := int64(float64(gains) * cfg.ExpRate)
	s.Exp += expGain
	leveled := false
	if s.Exp >= cfg.LevelUpExp {
		s.Level++
		s.Exp = 0
		leveled = true
	}

	s.LastTick = time.Now()
	return TickResult{
		Gains:   gains,
		Items:   items,
		RareEvt: rare,
		Level:   s.Level,
		CurExp:  s.Exp,
		Leveled: leveled,
	}
}
