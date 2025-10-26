package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/idle-server/common"
)

var Sequences map[string]*common.SequenceConfig
var extendedSequences map[string]*SequenceConfig // 存储扩展配置

// EquipmentDrop 装备掉落配置
type EquipmentDrop struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DropChance float64 `json:"drop_chance"`
	MinLevel   int     `json:"min_level"` // 最低序列等级要求
}

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

// SequenceConfig 扩展common.SequenceConfig，添加装备掉落信息和子项目
type SequenceConfig struct {
	common.SequenceConfig
	SubProjects    []common.SequenceSubProject `json:"sub_projects"`
	EquipmentDrops []EquipmentDrop             `json:"equipment_drops"`
}

func LoadConfig(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 解析配置
	if err := json.Unmarshal(b, &extendedSequences); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	// 同时创建common类型的配置供其他地方使用
	Sequences = make(map[string]*common.SequenceConfig)
	for id, cfg := range extendedSequences {
		Sequences[id] = &cfg.SequenceConfig
	}

	fmt.Printf("✅ loaded %d sequences from %s\n", len(Sequences), path)
	return nil
}

// GetAllSequences 返回所有可用序列（ID + Name），用于前端下拉选择
func GetSequenceSummaries() []SequenceSummary {
	briefs := make([]SequenceSummary, 0, len(extendedSequences))
	for id, cfg := range extendedSequences {
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
			TickInterval: int(cfg.TickInterval),
			SubProjects:  subs,
		})
	}
	// 按 ID 排序，确保稳定顺序，修复重连时序列错乱问题
	sort.Slice(briefs, func(i, j int) bool { return briefs[i].ID < briefs[j].ID })
	return briefs
}

func GetSequenceConfig(id string) (*SequenceConfig, bool) {
	c, ok := extendedSequences[id]
	return c, ok
}

// GetSubProject 根据ID查找子项目
func (c *SequenceConfig) GetSubProject(id string) (*common.SequenceSubProject, bool) {
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
