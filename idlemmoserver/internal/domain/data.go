package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

var Sequences map[string]*SequenceConfig

// SequenceSummary 向前端返回序列的完整概览
type SequenceSummary struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	TickInterval int               `json:"tick_interval"`
	SubProjects  []SubProjectBrief `json:"sub_projects"`
}

// SubProjectBrief 为前端提供展示信息
type SubProjectBrief struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	UnlockLevel    int     `json:"unlock_level"`
	Description    string  `json:"description"`
	GainMultiplier float64 `json:"gain_multiplier"`
	RareBonus      float64 `json:"rare_chance_bonus"`
	ExpMultiplier  float64 `json:"exp_multiplier"`
	IntervalMod    float64 `json:"interval_modifier"`
}

type SequenceConfig struct {
	Name         string               `json:"name"`
	BaseGain     int64                `json:"base_gain"`
	GrowthFactor float64              `json:"growth_factor"`
	TickInterval int                  `json:"tick_interval"`
	RareChance   float64              `json:"rare_chance"`
	Drops        []Item               `json:"drops"`
	RareEvents   []RareEvent          `json:"rare_events"`
	SubProjects  []SequenceSubProject `json:"sub_projects"`

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
func GetSequenceSummaries() []SequenceSummary {
	briefs := make([]SequenceSummary, 0, len(Sequences))
	for id, cfg := range Sequences {
		name := cfg.Name
		if name == "" {
			name = id
		}

		subs := make([]SubProjectBrief, 0, len(cfg.SubProjects))
		for _, sp := range cfg.SubProjects {
			subs = append(subs, SubProjectBrief{
				ID:             sp.ID,
				Name:           sp.Name,
				UnlockLevel:    sp.UnlockLevel,
				Description:    sp.Description,
				GainMultiplier: sp.GainMultiplier,
				RareBonus:      sp.RareChanceBonus,
				ExpMultiplier:  sp.ExpMultiplier,
				IntervalMod:    sp.IntervalModifier,
			})
		}

		briefs = append(briefs, SequenceSummary{
			ID:           id,
			Name:         name,
			TickInterval: cfg.TickInterval,
			SubProjects:  subs,
		})
	}
	// 按 ID 排序，确保稳定顺序，修复重连时序列错乱问题
	sort.Slice(briefs, func(i, j int) bool { return briefs[i].ID < briefs[j].ID })
	return briefs
}

func GetSequenceConfig(id string) (*SequenceConfig, bool) {
	c, ok := Sequences[id]
	return c, ok
}

// GetSubProject 根据ID查找子项目
func (c *SequenceConfig) GetSubProject(id string) (*SequenceSubProject, bool) {
	if id == "" {
		return nil, false
	}
	for i := range c.SubProjects {
		if c.SubProjects[i].ID == id {
			return &c.SubProjects[i], true
		}
	}
	return nil, false
}
