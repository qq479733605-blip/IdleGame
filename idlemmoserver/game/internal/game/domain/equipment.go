package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

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

type EquipmentBonus struct {
	GainMultiplier  float64 `json:"gain_multiplier"`
	RareChanceBonus float64 `json:"rare_chance_bonus"`
	ExpMultiplier   float64 `json:"exp_multiplier"`
}

type EquippedItem struct {
	Definition  EquipmentDefinition `json:"definition"`
	Enhancement int                 `json:"enhancement"`
}

type EquipmentLoadout struct {
	slots map[EquipmentSlot]*EquippedItem
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

type equipmentCatalogData map[string]EquipmentDefinition

var equipmentCatalog equipmentCatalogData

func LoadEquipmentConfig(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load equipment config: %w", err)
	}
	var data equipmentCatalogData
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("parse equipment config: %w", err)
	}
	equipmentCatalog = data
	return nil
}

func GetEquipmentDefinition(id string) (EquipmentDefinition, bool) {
	if equipmentCatalog == nil {
		return EquipmentDefinition{}, false
	}
	def, ok := equipmentCatalog[id]
	return def, ok
}

func GetEquipmentCatalogSummary() map[string]EquippedItemView {
	if equipmentCatalog == nil {
		return map[string]EquippedItemView{}
	}
	result := make(map[string]EquippedItemView, len(equipmentCatalog))
	// 按照键排序用于稳定输出
	keys := make([]string, 0, len(equipmentCatalog))
	for id := range equipmentCatalog {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		def := equipmentCatalog[id]
		result[id] = EquippedItemView{
			ItemID:      def.ID,
			Name:        def.Name,
			Slot:        def.Slot,
			Quality:     def.Quality,
			Description: def.Description,
			Enhancement: 0,
			Attributes:  def.Attributes,
		}
	}
	return result
}

func NewEquipmentLoadout() *EquipmentLoadout {
	return &EquipmentLoadout{slots: make(map[EquipmentSlot]*EquippedItem)}
}

func (l *EquipmentLoadout) Equip(def EquipmentDefinition, enhancement int) (replaced *EquippedItem) {
	if l.slots == nil {
		l.slots = make(map[EquipmentSlot]*EquippedItem)
	}
	replaced = l.slots[def.Slot]
	l.slots[def.Slot] = &EquippedItem{Definition: def, Enhancement: enhancement}
	return replaced
}

func (l *EquipmentLoadout) Unequip(slot EquipmentSlot) *EquippedItem {
	if l.slots == nil {
		return nil
	}
	item := l.slots[slot]
	delete(l.slots, slot)
	return item
}

func (l *EquipmentLoadout) Get(slot EquipmentSlot) *EquippedItem {
	if l.slots == nil {
		return nil
	}
	return l.slots[slot]
}

func (l *EquipmentLoadout) TotalBonus() EquipmentBonus {
	bonus := EquipmentBonus{}
	if l.slots == nil {
		return bonus
	}
	for _, item := range l.slots {
		bonus.GainMultiplier += item.Definition.Attributes.GainMultiplier
		bonus.RareChanceBonus += item.Definition.Attributes.RareChanceBonus
		bonus.ExpMultiplier += item.Definition.Attributes.ExpMultiplier
	}
	return bonus
}

func (l *EquipmentLoadout) Export() map[string]EquippedItemView {
	view := make(map[string]EquippedItemView)
	if l.slots == nil {
		return view
	}
	for slot, item := range l.slots {
		view[string(slot)] = EquippedItemView{
			ItemID:      item.Definition.ID,
			Name:        item.Definition.Name,
			Slot:        slot,
			Quality:     item.Definition.Quality,
			Description: item.Definition.Description,
			Enhancement: item.Enhancement,
			Attributes:  item.Definition.Attributes,
		}
	}
	return view
}

func (l *EquipmentLoadout) ExportState() map[string]EquipmentState {
	state := make(map[string]EquipmentState)
	if l.slots == nil {
		return state
	}
	for slot, item := range l.slots {
		state[string(slot)] = EquipmentState{ItemID: item.Definition.ID, Enhancement: item.Enhancement}
	}
	return state
}

func (l *EquipmentLoadout) ImportState(state map[string]EquipmentState) {
	if l.slots == nil {
		l.slots = make(map[EquipmentSlot]*EquippedItem)
	}
	for slot, st := range state {
		def, ok := GetEquipmentDefinition(st.ItemID)
		if !ok {
			continue
		}
		l.slots[EquipmentSlot(slot)] = &EquippedItem{Definition: def, Enhancement: st.Enhancement}
	}
}
