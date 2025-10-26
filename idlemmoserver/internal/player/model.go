package player

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"idlemmoserver/internal/common"
)

type Inventory struct {
	mu    sync.Mutex
	slots map[string]int64
	limit int
}

func NewInventory(limit int) *Inventory {
	return &Inventory{slots: make(map[string]int64), limit: limit}
}

func (inv *Inventory) AddItem(item common.ItemDrop, count int64) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if _, ok := inv.slots[item.ID]; !ok && len(inv.slots) >= inv.limit {
		return fmt.Errorf("inventory full")
	}
	inv.slots[item.ID] += count
	return nil
}

func (inv *Inventory) RemoveItem(itemID string, count int64) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if inv.slots[itemID] < count {
		return fmt.Errorf("not enough items")
	}
	inv.slots[itemID] -= count
	if inv.slots[itemID] <= 0 {
		delete(inv.slots, itemID)
	}
	return nil
}

func (inv *Inventory) List() map[string]int64 {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	out := make(map[string]int64, len(inv.slots))
	for k, v := range inv.slots {
		out[k] = v
	}
	return out
}

func (inv *Inventory) Import(snapshot common.InventorySnapshot) {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	inv.slots = make(map[string]int64, len(snapshot))
	for k, v := range snapshot {
		inv.slots[k] = v
	}
}

type EquipmentLoadout struct {
	slots map[common.EquipmentSlot]*EquippedItem
}

type EquippedItem struct {
	Definition  common.EquipmentDefinition
	Enhancement int
}

func NewEquipmentLoadout() *EquipmentLoadout {
	return &EquipmentLoadout{slots: make(map[common.EquipmentSlot]*EquippedItem)}
}

func (l *EquipmentLoadout) Equip(def common.EquipmentDefinition, enhancement int) (replaced *EquippedItem) {
	if l.slots == nil {
		l.slots = make(map[common.EquipmentSlot]*EquippedItem)
	}
	replaced = l.slots[def.Slot]
	l.slots[def.Slot] = &EquippedItem{Definition: def, Enhancement: enhancement}
	return replaced
}

func (l *EquipmentLoadout) Unequip(slot common.EquipmentSlot) *EquippedItem {
	if l.slots == nil {
		return nil
	}
	item := l.slots[slot]
	delete(l.slots, slot)
	return item
}

func (l *EquipmentLoadout) Get(slot common.EquipmentSlot) *EquippedItem {
	if l.slots == nil {
		return nil
	}
	return l.slots[slot]
}

func (l *EquipmentLoadout) TotalBonus() common.EquipmentBonus {
	bonus := common.EquipmentBonus{}
	for _, item := range l.slots {
		bonus.GainMultiplier += item.Definition.Attributes.GainMultiplier
		bonus.RareChanceBonus += item.Definition.Attributes.RareChanceBonus
		bonus.ExpMultiplier += item.Definition.Attributes.ExpMultiplier
	}
	return bonus
}

func (l *EquipmentLoadout) ExportState() map[string]common.EquipmentState {
	state := make(map[string]common.EquipmentState)
	for slot, item := range l.slots {
		state[string(slot)] = common.EquipmentState{ItemID: item.Definition.ID, Enhancement: item.Enhancement}
	}
	return state
}

func (l *EquipmentLoadout) ImportState(state map[string]common.EquipmentState) {
	if l.slots == nil {
		l.slots = make(map[common.EquipmentSlot]*EquippedItem)
	}
	for slot, st := range state {
		def, ok := GetEquipmentDefinition(st.ItemID)
		if !ok {
			continue
		}
		l.slots[common.EquipmentSlot(slot)] = &EquippedItem{Definition: def, Enhancement: st.Enhancement}
	}
}

func (l *EquipmentLoadout) ExportView() map[string]common.EquippedItemView {
	view := make(map[string]common.EquippedItemView)
	for slot, item := range l.slots {
		view[string(slot)] = common.EquippedItemView{
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

type equipmentCatalogData map[string]common.EquipmentDefinition

var (
	equipmentCatalog equipmentCatalogData
	catalogOnce      sync.Once
)

func LoadEquipmentConfig(path string) error {
	var err error
	catalogOnce.Do(func() {
		err = loadEquipment(path)
	})
	return err
}

func loadEquipment(path string) error {
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

func GetEquipmentDefinition(id string) (common.EquipmentDefinition, bool) {
	if equipmentCatalog == nil {
		return common.EquipmentDefinition{}, false
	}
	def, ok := equipmentCatalog[id]
	return def, ok
}

func GetEquipmentCatalogSummary() map[string]common.EquippedItemView {
	if equipmentCatalog == nil {
		return map[string]common.EquippedItemView{}
	}
	result := make(map[string]common.EquippedItemView, len(equipmentCatalog))
	keys := make([]string, 0, len(equipmentCatalog))
	for id := range equipmentCatalog {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		def := equipmentCatalog[id]
		result[id] = common.EquippedItemView{
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

type State struct {
	PlayerID         string
	SeqLevels        map[string]int
	Inventory        *Inventory
	Equipment        *EquipmentLoadout
	Exp              int64
	OfflineLimit     time.Duration
	IsOnline         bool
	OfflineStart     time.Time
	LastActive       time.Time
	ActiveSubProject string
}

func NewState(playerID string) *State {
	return &State{
		PlayerID:     playerID,
		SeqLevels:    make(map[string]int),
		Inventory:    NewInventory(200),
		Equipment:    NewEquipmentLoadout(),
		OfflineLimit: 10 * time.Hour,
		IsOnline:     true,
	}
}

func (s *State) Snapshot() common.PlayerSnapshot {
	return common.PlayerSnapshot{
		PlayerID:          s.PlayerID,
		SeqLevels:         s.SeqLevels,
		Inventory:         s.Inventory.List(),
		Exp:               s.Exp,
		Equipment:         s.Equipment.ExportState(),
		OfflineLimitHours: int64(s.OfflineLimit / time.Hour),
	}
}

func (s *State) ApplySnapshot(snapshot *common.PlayerSnapshot) {
	if snapshot == nil {
		return
	}
	s.SeqLevels = snapshot.SeqLevels
	s.Inventory.Import(snapshot.Inventory)
	s.Exp = snapshot.Exp
	s.Equipment.ImportState(snapshot.Equipment)
	if snapshot.OfflineLimitHours > 0 {
		s.OfflineLimit = time.Duration(snapshot.OfflineLimitHours) * time.Hour
	}
}

func (s *State) SetOnline(connBound bool) {
	s.IsOnline = connBound
	if connBound {
		s.OfflineStart = time.Time{}
	} else {
		s.OfflineStart = time.Now()
	}
	s.LastActive = time.Now()
}
