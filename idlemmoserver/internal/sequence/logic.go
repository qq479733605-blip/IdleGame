package sequence

import (
	"math/rand"
	"time"

	"idlemmoserver/internal/common"
)

type Sequence struct {
	ID         string
	Level      int
	Exp        int64
	StartTime  time.Time
	LastTick   time.Time
	SubProject *SubProject
}

type TickResult struct {
	Gains        int64
	Items        []common.ItemDrop
	RareEvt      *common.RareEvent
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

func NewSequence(seqID string, level int, sub *SubProject) *Sequence {
	s := &Sequence{ID: seqID, Level: level, StartTime: time.Now(), LastTick: time.Now()}
	if sub != nil {
		s.SubProject = sub
	}
	return s
}

func (s *Sequence) Tick(cfg *Config, bonus common.EquipmentBonus) TickResult {
	drops := make([]common.ItemDrop, len(cfg.Drops))
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

	gains := cfg.BaseGain + int64(float64(s.Level)*cfg.GrowthFactor)
	gains = int64(float64(gains) * gainMultiplier)
	if gains < 0 {
		gains = 0
	}

	items := []common.ItemDrop{}
	for _, it := range drops {
		if rand.Float64() < it.DropChance {
			items = append(items, it)
		}
	}

	for _, equip := range cfg.EquipmentDrops {
		if s.Level >= equip.MinLevel && rand.Float64() < equip.DropChance {
			items = append(items, common.ItemDrop{
				ID:          equip.ID,
				Name:        equip.Name,
				DropChance:  equip.DropChance,
				Value:       50,
				IsEquipment: true,
			})
		}
	}

	var rare *common.RareEvent
	if rand.Float64() < rareChance && len(cfg.RareEvents) > 0 {
		r := cfg.RareEvents[rand.Intn(len(cfg.RareEvents))]
		rare = &r
		gains = int64(float64(gains) * rare.MultGain)
	}

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

func (s *Sequence) EffectiveInterval(cfg *Config) time.Duration {
	base := float64(cfg.TickInterval)
	if base <= 0 {
		base = 1
	}
	if s.SubProject != nil && s.SubProject.IntervalModifier > 0 {
		base = base * s.SubProject.IntervalModifier
	}
	if base < 0.5 {
		base = 0.5
	}
	return time.Duration(base * float64(time.Second))
}

func EffectiveInterval(cfg *Config, sub *SubProject) time.Duration {
	base := float64(cfg.TickInterval)
	if base <= 0 {
		base = 1
	}
	if sub != nil && sub.IntervalModifier > 0 {
		base = base * sub.IntervalModifier
	}
	if base < 0.5 {
		base = 0.5
	}
	return time.Duration(base * float64(time.Second))
}
