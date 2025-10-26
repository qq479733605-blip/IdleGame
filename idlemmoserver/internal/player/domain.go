package player

import (
	"math/rand"
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"
	"idlemmoserver/internal/sequence"
)

type Domain struct {
	state *State
}

func NewDomain(state *State) *Domain {
	return &Domain{state: state}
}

func (d *Domain) State() *State {
	return d.state
}

func (d *Domain) EnsureSequenceDefaults() {
	if len(d.state.SeqLevels) == 0 {
		d.state.SeqLevels = make(map[string]int)
		for id := range sequence.AllConfigs() {
			d.state.SeqLevels[id] = 1
		}
	}
}

func (d *Domain) ApplySnapshot(snapshot *common.PlayerSnapshot) {
	if snapshot == nil {
		d.EnsureSequenceDefaults()
		return
	}
	d.state.ApplySnapshot(snapshot)
	d.EnsureSequenceDefaults()
}

func (d *Domain) PrepareLoadResponse() map[string]any {
	return map[string]any{
		"type":                "S_LoadOK",
		"exp":                 d.state.Exp,
		"bag":                 d.state.Inventory.List(),
		"offline_limit_hours": int64(d.state.OfflineLimit / time.Hour),
		"equipment":           d.state.Equipment.ExportView(),
		"equipment_bonus":     d.state.Equipment.TotalBonus(),
	}
}

func (d *Domain) PrepareNewPlayerResponse() map[string]any {
	d.EnsureSequenceDefaults()
	return map[string]any{"type": "S_NewPlayer"}
}

func (d *Domain) OfflineRewards() common.OfflineReward {
	if d.state.OfflineStart.IsZero() {
		return common.OfflineReward{Duration: 0, Gains: 0, Items: map[string]int64{}}
	}
	duration := time.Since(d.state.OfflineStart)
	if duration <= 0 || duration >= d.state.OfflineLimit {
		return common.OfflineReward{Duration: duration, Gains: 0, Items: map[string]int64{}}
	}

	gains := int64(0)
	items := make(map[string]int64)
	seconds := duration.Seconds()

	for seqID, level := range d.state.SeqLevels {
		if level <= 0 {
			continue
		}
		cfg, exists := sequence.GetConfig(seqID)
		if !exists || cfg == nil {
			continue
		}
		interval := cfg.TickInterval
		if interval <= 0 {
			interval = 1
		}
		ticks := int64(seconds / float64(interval))
		if ticks <= 0 {
			continue
		}
		gain := cfg.BaseGain + int64(float64(level)*cfg.GrowthFactor)
		gains += gain * ticks

		for _, drop := range cfg.Drops {
			if drop.DropChance <= 0 {
				continue
			}
			expected := float64(ticks) * drop.DropChance
			guaranteed := int64(expected)
			remainder := expected - float64(guaranteed)
			count := guaranteed
			if rand.Float64() < remainder {
				count++
			}
			if count > 0 {
				items[drop.ID] += count
			}
		}
	}

	return common.OfflineReward{Duration: duration, Gains: gains, Items: items}
}

func (d *Domain) BuildReconnectPayload(currentSeqID string, currentSeqLevel int, isRunning bool) map[string]any {
	return map[string]any{
		"type":               "S_Reconnected",
		"msg":                "重连成功",
		"seq_id":             currentSeqID,
		"seq_level":          currentSeqLevel,
		"exp":                d.state.Exp,
		"bag":                d.state.Inventory.List(),
		"is_running":         isRunning,
		"seq_levels":         d.state.SeqLevels,
		"equipment":          d.state.Equipment.ExportView(),
		"equipment_bonus":    d.state.Equipment.TotalBonus(),
		"active_sub_project": d.state.ActiveSubProject,
	}
}

func (d *Domain) ApplySequenceResult(res *common.MsgSequenceResult) common.SequenceResultPayload {
	logx.Info("Player sequence result", "player", res.PlayerID, "seq", res.SeqID, "gains", res.Gains)

	for _, item := range res.Items {
		if err := d.state.Inventory.AddItem(item, 1); err != nil {
			logx.Error("add item failed", "item", item.ID, "err", err)
		}
	}
	if res.SeqID != "" {
		d.state.SeqLevels[res.SeqID] = res.Level
	}
	d.state.ActiveSubProject = res.SubProjectID
	d.state.Exp += res.Gains

	return common.SequenceResultPayload{
		PlayerID:     res.PlayerID,
		SeqID:        res.SeqID,
		Items:        res.Items,
		Rare:         res.Rare,
		Gains:        res.Gains,
		Level:        res.Level,
		CurExp:       res.CurExp,
		Leveled:      res.Leveled,
		SubProjectID: res.SubProjectID,
		Bonus:        d.state.Equipment.TotalBonus(),
		Bag:          d.state.Inventory.List(),
	}
}

func (d *Domain) UseItem(itemID string, count int64) (int64, error) {
	if count <= 0 {
		return 0, ErrInvalidItemCount
	}
	if err := d.state.Inventory.RemoveItem(itemID, count); err != nil {
		return 0, err
	}
	gain := count * 10
	d.state.Exp += gain
	return gain, nil
}

func (d *Domain) RemoveItem(itemID string, count int64) error {
	return d.state.Inventory.RemoveItem(itemID, count)
}

func (d *Domain) EquipItem(itemID string, enhancement int) (*EquippedItem, error) {
	def, ok := GetEquipmentDefinition(itemID)
	if !ok {
		return nil, ErrItemNotEquippable
	}
	if err := d.state.Inventory.RemoveItem(itemID, 1); err != nil {
		return nil, err
	}
	replaced := d.state.Equipment.Equip(def, enhancement)
	return replaced, nil
}

func (d *Domain) RestoreEquippedItem(item common.ItemDrop) {
	if err := d.state.Inventory.AddItem(item, 1); err != nil {
		logx.Warn("restore item failed", "item", item.ID, "err", err)
	}
}

func (d *Domain) Unequip(slot common.EquipmentSlot) *EquippedItem {
	return d.state.Equipment.Unequip(slot)
}
