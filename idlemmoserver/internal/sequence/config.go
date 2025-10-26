package sequence

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"idlemmoserver/internal/common"
)

type EquipmentDrop struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DropChance float64 `json:"drop_chance"`
	MinLevel   int     `json:"min_level"`
}

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

type Config struct {
	Name           string             `json:"name"`
	BaseGain       int64              `json:"base_gain"`
	GrowthFactor   float64            `json:"growth_factor"`
	TickInterval   int                `json:"tick_interval"`
	RareChance     float64            `json:"rare_chance"`
	Drops          []common.ItemDrop  `json:"drops"`
	EquipmentDrops []EquipmentDrop    `json:"equipment_drops"`
	RareEvents     []common.RareEvent `json:"rare_events"`
	SubProjects    []SubProject       `json:"sub_projects"`
	LevelUpExp     int64              `json:"levelup_exp"`
	ExpRate        float64            `json:"exp_rate"`
}

type SubProject struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	UnlockLevel      int               `json:"unlock_level"`
	Description      string            `json:"description"`
	GainMultiplier   float64           `json:"gain_multiplier"`
	RareChanceBonus  float64           `json:"rare_chance_bonus"`
	ExpMultiplier    float64           `json:"exp_multiplier"`
	IntervalModifier float64           `json:"interval_modifier"`
	ExtraDrops       []common.ItemDrop `json:"extra_drops"`
}

type Summary struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	TickInterval int               `json:"tick_interval"`
	SubProjects  []SubProjectBrief `json:"sub_projects"`
}

var sequences map[string]*Config

func LoadConfig(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	var data map[string]*Config
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	sequences = data
	return nil
}

func GetConfig(id string) (*Config, bool) {
	cfg, ok := sequences[id]
	return cfg, ok
}

func AllConfigs() map[string]*Config {
	return sequences
}

func GetSummaries() []Summary {
	briefs := make([]Summary, 0, len(sequences))
	for id, cfg := range sequences {
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
		briefs = append(briefs, Summary{ID: id, Name: name, TickInterval: cfg.TickInterval, SubProjects: subs})
	}
	sort.Slice(briefs, func(i, j int) bool { return briefs[i].ID < briefs[j].ID })
	return briefs
}

func (c *Config) GetSubProject(id string) (*SubProject, bool) {
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
