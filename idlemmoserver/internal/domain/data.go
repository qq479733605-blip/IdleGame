package domain

import (
	"encoding/json"
	"fmt"
	"os"
)

var Sequences map[string]*SequenceConfig

// SequenceConfig 对应 JSON 配置的结构
type SequenceConfig struct {
	Name         string      `json:"name"`
	BaseGain     int64       `json:"base_gain"`
	GrowthFactor float64     `json:"growth_factor"`
	TickInterval int         `json:"tick_interval"`
	RareChance   float64     `json:"rare_chance"`
	Drops        []Item      `json:"drops"`
	RareEvents   []RareEvent `json:"rare_events"`
}

// LoadConfig 从 JSON 文件加载配置
func LoadConfig(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := json.Unmarshal(file, &Sequences); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	fmt.Printf("✅ loaded %d sequences from %s\n", len(Sequences), path)
	return nil
}

// GetSequenceConfig 获取指定序列配置
func GetSequenceConfig(id string) (*SequenceConfig, bool) {
	c, ok := Sequences[id]
	return c, ok
}
