package domain

import (
	"fmt"
	"sync"
)

type Inventory struct {
	mu    sync.Mutex
	Slots map[string]int64 // itemID -> count
	Limit int              // 最大种类数量
}

func NewInventory(limit int) *Inventory {
	return &Inventory{
		Slots: make(map[string]int64),
		Limit: limit,
	}
}

func (inv *Inventory) AddItem(item Item, count int64) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if _, ok := inv.Slots[item.ID]; !ok && len(inv.Slots) >= inv.Limit {
		return fmt.Errorf("inventory full")
	}
	inv.Slots[item.ID] += count
	return nil
}

func (inv *Inventory) RemoveItem(itemID string, count int64) error {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if inv.Slots[itemID] < count {
		return fmt.Errorf("not enough items")
	}
	inv.Slots[itemID] -= count
	if inv.Slots[itemID] <= 0 {
		delete(inv.Slots, itemID)
	}
	return nil
}

func (inv *Inventory) List() map[string]int64 {
	inv.mu.Lock()
	defer inv.mu.Unlock()
	copy := make(map[string]int64, len(inv.Slots))
	for k, v := range inv.Slots {
		copy[k] = v
	}
	return copy
}
