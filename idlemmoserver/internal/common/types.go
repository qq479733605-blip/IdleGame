package common

import "time"

type ItemDrop struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DropChance  float64 `json:"drop_chance"`
	Value       int64   `json:"value"`
	IsEquipment bool    `json:"is_equipment"`
}

type RareEvent struct {
	Name     string  `json:"name"`
	Effect   string  `json:"effect"`
	MultGain float64 `json:"mult_gain"`
}

type ItemQuality string

type EquipmentSlot string

const (
	QualityCommon    ItemQuality = "common"
	QualityUncommon  ItemQuality = "uncommon"
	QualityRare      ItemQuality = "rare"
	QualityEpic      ItemQuality = "epic"
	QualityLegendary ItemQuality = "legendary"
	QualityMythic    ItemQuality = "mythic"

	SlotWeapon EquipmentSlot = "weapon"
	SlotArmor  EquipmentSlot = "armor"
	SlotHead   EquipmentSlot = "head"
	SlotHand   EquipmentSlot = "hand"
	SlotFoot   EquipmentSlot = "foot"
	SlotRelic  EquipmentSlot = "relic"
)

type EquipmentAttributes struct {
	GainMultiplier  float64 `json:"gain_multiplier"`
	RareChanceBonus float64 `json:"rare_chance_bonus"`
	ExpMultiplier   float64 `json:"exp_multiplier"`
}

type EquipmentDefinition struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Slot        EquipmentSlot       `json:"slot"`
	Quality     ItemQuality         `json:"quality"`
	Description string              `json:"description"`
	Attributes  EquipmentAttributes `json:"attributes"`
}

type EquipmentState struct {
	ItemID      string `json:"item_id"`
	Enhancement int    `json:"enhancement"`
}

type EquippedItemView struct {
	ItemID      string              `json:"item_id"`
	Name        string              `json:"name"`
	Slot        EquipmentSlot       `json:"slot"`
	Quality     ItemQuality         `json:"quality"`
	Description string              `json:"description"`
	Enhancement int                 `json:"enhancement"`
	Attributes  EquipmentAttributes `json:"attributes"`
}

type EquipmentBonus struct {
	GainMultiplier  float64 `json:"gain_multiplier"`
	RareChanceBonus float64 `json:"rare_chance_bonus"`
	ExpMultiplier   float64 `json:"exp_multiplier"`
}

type InventorySnapshot map[string]int64

type PlayerSnapshot struct {
	PlayerID          string                    `json:"player_id"`
	SeqLevels         map[string]int            `json:"seq_levels"`
	Inventory         InventorySnapshot         `json:"inventory"`
	Exp               int64                     `json:"exp"`
	Equipment         map[string]EquipmentState `json:"equipment"`
	OfflineLimitHours int64                     `json:"offline_limit_hours"`
}

type SequenceResultPayload struct {
	PlayerID     string           `json:"player_id"`
	SeqID        string           `json:"seq_id"`
	Items        []ItemDrop       `json:"items"`
	Rare         []string         `json:"rare"`
	Gains        int64            `json:"gains"`
	Level        int              `json:"level"`
	CurExp       int64            `json:"cur_exp"`
	Leveled      bool             `json:"leveled"`
	SubProjectID string           `json:"sub_project_id"`
	Bonus        EquipmentBonus   `json:"equipment_bonus"`
	Bag          map[string]int64 `json:"bag"`
}

type OfflineReward struct {
	Duration time.Duration
	Gains    int64
	Items    map[string]int64
}
