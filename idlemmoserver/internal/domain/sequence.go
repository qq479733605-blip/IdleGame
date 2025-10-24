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
}

func (s *Sequence) Tick(cfg *SequenceConfig) TickResult {
	gains := cfg.BaseGain + int64(float64(s.Level)*cfg.GrowthFactor)

	items := []Item{}
	for _, item := range cfg.Drops {
		if rand.Float64() < item.DropChance {
			items = append(items, item)
		}
	}

	var rare *RareEvent
	if rand.Float64() < cfg.RareChance && len(cfg.RareEvents) > 0 {
		rare = &cfg.RareEvents[rand.Intn(len(cfg.RareEvents))]
		gains = int64(float64(gains) * rare.MultGain)
	}

	s.Exp += gains
	s.LastTick = time.Now()

	return TickResult{Gains: gains, Items: items, RareEvt: rare}
}
