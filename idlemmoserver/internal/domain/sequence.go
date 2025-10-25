package domain

import (
	"math/rand"
	"time"
)

type SequenceSubProject struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	UnlockLevel      int     `json:"unlock_level"`
	Description      string  `json:"description"`
	GainMultiplier   float64 `json:"gain_multiplier"`
	RareChanceBonus  float64 `json:"rare_chance_bonus"`
	ExpMultiplier    float64 `json:"exp_multiplier"`
	IntervalModifier float64 `json:"interval_modifier"`
	ExtraDrops       []Item  `json:"extra_drops"`
}

type Sequence struct {
	ID         string
	Level      int
	Exp        int64
	StartTime  time.Time
	LastTick   time.Time
	SubProject *SequenceSubProject
}

type TickResult struct {
	Gains        int64
	Items        []Item
	RareEvt      *RareEvent
	Level        int   // 当前等级（Tick 后）
	CurExp       int64 // 当前经验（Tick 后）
	Leveled      bool  // 是否在本次升级
	SubProjectID string
}

func (s *Sequence) Tick(cfg *SequenceConfig, bonus EquipmentBonus) TickResult {
	// 复制掉落配置，避免修改原数据
	drops := make([]Item, len(cfg.Drops))
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
	items := []Item{}
	for _, it := range drops {
		if rand.Float64() < it.DropChance {
			items = append(items, it)
		}
	}

	// 奇遇
	var rare *RareEvent
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
	return TickResult{
		Gains:        gains,
		Items:        items,
		RareEvt:      rare,
		Level:        s.Level,
		CurExp:       s.Exp,
		Leveled:      leveled,
		SubProjectID: subID,
	}
}

func (s *Sequence) SetSubProject(sp *SequenceSubProject) {
	s.SubProject = sp
}

func (cfg *SequenceConfig) EffectiveInterval(sp *SequenceSubProject) time.Duration {
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
