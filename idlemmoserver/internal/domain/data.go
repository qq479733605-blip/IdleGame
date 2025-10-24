package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

var Sequences map[string]*SequenceConfig

// SeqBrief 用于向前端返回可选序列的简要信息
type SeqBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SequenceConfig struct {
	Name         string      `json:"name"`
	BaseGain     int64       `json:"base_gain"`
	GrowthFactor float64     `json:"growth_factor"`
	TickInterval int         `json:"tick_interval"`
	RareChance   float64     `json:"rare_chance"`
	Drops        []Item      `json:"drops"`
	RareEvents   []RareEvent `json:"rare_events"`

	// 成长相关（表驱动）
	LevelUpExp int64   `json:"levelup_exp"`
	ExpRate    float64 `json:"exp_rate"` // 收益 * ExpRate → 每次获得的经验
}

func LoadConfig(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := json.Unmarshal(b, &Sequences); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	fmt.Printf("✅ loaded %d sequences from %s\n", len(Sequences), path)
	return nil
}

// GetAllSequences 返回所有可用序列（ID + Name），用于前端下拉选择
func GetAllSequences() []SeqBrief {
	briefs := make([]SeqBrief, 0, len(Sequences))
	for id, cfg := range Sequences {
		name := cfg.Name
		if name == "" {
			name = id
		}
		briefs = append(briefs, SeqBrief{ID: id, Name: name})
	}
	// 按 ID 排序，确保稳定顺序，修复重连时序列错乱问题
	sort.Slice(briefs, func(i, j int) bool { return briefs[i].ID < briefs[j].ID })
	return briefs
}

func GetSequenceConfig(id string) (*SequenceConfig, bool) {
	c, ok := Sequences[id]
	return c, ok
}
