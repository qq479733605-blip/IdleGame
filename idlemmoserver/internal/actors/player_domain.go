package actors

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	gamedomain "idlemmoserver/internal/domain"
	"idlemmoserver/internal/persist"
)

// PlayerDomain 封装所有与玩家相关的核心业务逻辑，实现 DDD 中的领域服务概念。
type PlayerDomain struct{}

// NewPlayerDomain 创建 PlayerDomain 实例，目前不包含额外状态。
func NewPlayerDomain() *PlayerDomain {
	return &PlayerDomain{}
}

var (
	// ErrSequenceNotFound 表示请求的修炼序列不存在。
	ErrSequenceNotFound = errors.New("未找到对应的修炼序列")
	// ErrSubProjectNotFound 表示请求的子项目不存在。
	ErrSubProjectNotFound = errors.New("未找到对应的子项目")
	// ErrSubProjectLocked 表示子项目尚未解锁。
	ErrSubProjectLocked = errors.New("子项目未解锁")
)

// InitNewPlayer 为新玩家初始化默认数据，例如默认序列等级。
func (d *PlayerDomain) InitNewPlayer(model *PlayerModel) {
	model.SeqLevels = make(map[string]int)
	for seqID := range gamedomain.Sequences {
		model.SeqLevels[seqID] = 1
	}
}

// ApplyLoadedData 将持久化层的数据加载到领域模型中。
func (d *PlayerDomain) ApplyLoadedData(model *PlayerModel, data *persist.PlayerData) {
	if data == nil {
		d.InitNewPlayer(model)
		return
	}
	if data.SeqLevels != nil {
		model.SeqLevels = data.SeqLevels
	} else {
		d.InitNewPlayer(model)
	}
	model.Exp = data.Exp
	model.Inventory = gamedomain.NewInventory(200)
	if data.Inventory != nil {
		for id, count := range data.Inventory {
			_ = model.Inventory.AddItem(gamedomain.Item{ID: id, Name: id}, count)
		}
	}
	model.Equipment = gamedomain.NewEquipmentLoadout()
	if data.Equipment != nil {
		model.Equipment.ImportState(data.Equipment)
	}
	if data.OfflineLimitHours > 0 {
		model.SetOfflineLimitHours(data.OfflineLimitHours)
	}
}

// BuildReconnectedPayload 构造重连时需要同步到客户端的状态信息。
func (d *PlayerDomain) BuildReconnectedPayload(model *PlayerModel, isRunning bool) map[string]any {
	return map[string]any{
		"type":               "S_Reconnected",
		"msg":                "重连成功",
		"seq_id":             model.CurrentSeqID,
		"seq_level":          model.CurrentSeqLevel(),
		"exp":                model.Exp,
		"bag":                model.Inventory.List(),
		"is_running":         isRunning,
		"seq_levels":         model.SeqLevels,
		"equipment":          model.Equipment.Export(),
		"equipment_bonus":    model.Equipment.TotalBonus(),
		"active_sub_project": model.ActiveSubProject,
	}
}

// PrepareSequenceStart 校验修炼序列启动前的条件，并返回相关配置与等级信息。
func (d *PlayerDomain) PrepareSequenceStart(model *PlayerModel, seqID string, subProjectID string) (*gamedomain.SequenceConfig, *gamedomain.SequenceSubProject, int, error) {
	cfg, exists := gamedomain.GetSequenceConfig(seqID)
	if !exists || cfg == nil {
		return nil, nil, 0, ErrSequenceNotFound
	}
	level := model.SeqLevels[seqID]
	var subProject *gamedomain.SequenceSubProject
	if subProjectID != "" {
		sp, ok := cfg.GetSubProject(subProjectID)
		if !ok {
			return nil, nil, level, ErrSubProjectNotFound
		}
		if level < sp.UnlockLevel {
			return nil, nil, level, ErrSubProjectLocked
		}
		subProject = sp
	}
	return cfg, subProject, level, nil
}

// OnSequenceStarted 更新模型中的修炼状态。
func (d *PlayerDomain) OnSequenceStarted(model *PlayerModel, seqID string, subProject *gamedomain.SequenceSubProject) {
	subID := ""
	if subProject != nil {
		subID = subProject.ID
	}
	model.SetSequenceRunning(seqID, subID)
}

// OnSequenceStopped 清除模型中的修炼状态。
func (d *PlayerDomain) OnSequenceStopped(model *PlayerModel) {
	model.ClearRunningSequence()
}

// ApplySequenceResult 根据修炼结算结果更新玩家数据，并返回当前背包快照。
func (d *PlayerDomain) ApplySequenceResult(model *PlayerModel, res *SeqResult) map[string]int64 {
	for _, it := range res.Items {
		if err := model.Inventory.AddItem(it, 1); err != nil {
			// 背包空间不足时记录错误，但不阻断其他奖励。
			fmt.Printf("add item failed: %v\n", err)
		}
	}
	if res.SeqID != "" {
		model.SeqLevels[res.SeqID] = res.Level
	}
	model.ActiveSubProject = res.SubProjectID
	model.Exp += res.Gains
	return model.Inventory.List()
}

// ApplyOfflineRewards 结算离线收益，并将奖励写入模型。
func (d *PlayerDomain) ApplyOfflineRewards(model *PlayerModel, now time.Time) (int64, map[string]int64, time.Duration) {
	if model.OfflineStart.IsZero() {
		return 0, map[string]int64{}, 0
	}
	duration := now.Sub(model.OfflineStart)
	if duration <= 0 || (model.OfflineLimit > 0 && duration >= model.OfflineLimit) {
		return 0, map[string]int64{}, duration
	}
	gains := int64(0)
	items := make(map[string]int64)
	seconds := duration.Seconds()
	for seqID, level := range model.SeqLevels {
		if level <= 0 {
			continue
		}
		cfg, exists := gamedomain.GetSequenceConfig(seqID)
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
	if gains > 0 {
		model.Exp += gains
	}
	for itemID, count := range items {
		_ = model.Inventory.AddItem(gamedomain.Item{ID: itemID, Name: itemID}, count)
	}
	model.ResetOffline()
	return gains, items, duration
}

// EquipItem 负责穿戴装备，并处理原装备的回收与错误恢复。
func (d *PlayerDomain) EquipItem(model *PlayerModel, itemID string, enhancement int) error {
	def, ok := gamedomain.GetEquipmentDefinition(itemID)
	if !ok {
		return fmt.Errorf("该物品无法装备")
	}
	if err := model.Inventory.RemoveItem(itemID, 1); err != nil {
		return err
	}
	replaced := model.Equipment.Equip(def, enhancement)
	if replaced != nil {
		if err := model.Inventory.AddItem(gamedomain.Item{ID: replaced.Definition.ID, Name: replaced.Definition.Name}, 1); err != nil {
			model.Equipment.Equip(replaced.Definition, replaced.Enhancement)
			_ = model.Inventory.AddItem(gamedomain.Item{ID: def.ID, Name: def.Name}, 1)
			return fmt.Errorf("背包空间不足")
		}
	}
	return nil
}

// UnequipItem 卸下指定槽位的装备，并放回背包。
func (d *PlayerDomain) UnequipItem(model *PlayerModel, slot gamedomain.EquipmentSlot) error {
	item := model.Equipment.Unequip(slot)
	if item == nil {
		return fmt.Errorf("该位置没有装备")
	}
	if err := model.Inventory.AddItem(gamedomain.Item{ID: item.Definition.ID, Name: item.Definition.Name}, 1); err != nil {
		model.Equipment.Equip(item.Definition, item.Enhancement)
		return err
	}
	return nil
}

// UseItem 使用背包物品，这里暂时实现为固定增加经验。
func (d *PlayerDomain) UseItem(model *PlayerModel, itemID string, count int64) (int64, error) {
	if count <= 0 {
		return 0, fmt.Errorf("数量必须大于 0")
	}
	if err := model.Inventory.RemoveItem(itemID, count); err != nil {
		return 0, err
	}
	gain := count * 10
	model.Exp += gain
	return gain, nil
}

// RemoveItem 直接删除背包物品。
func (d *PlayerDomain) RemoveItem(model *PlayerModel, itemID string, count int64) error {
	return model.Inventory.RemoveItem(itemID, count)
}

// ShouldExpire 判断玩家离线是否超过上限，需要从 Actor 中触发销毁流程。
func (d *PlayerDomain) ShouldExpire(model *PlayerModel, now time.Time) bool {
	if model.IsOnline || model.OfflineStart.IsZero() {
		return false
	}
	if model.OfflineLimit <= 0 {
		return false
	}
	return now.Sub(model.OfflineStart) > model.OfflineLimit
}

// BuildLoginPayload 构造登录成功时返回的数据。
func (d *PlayerDomain) BuildLoginPayload(model *PlayerModel, isRunning bool) map[string]any {
	return map[string]any{
		"type":               "S_LoginOK",
		"msg":                "登录成功",
		"playerId":           model.ID,
		"exp":                model.Exp,
		"seq_levels":         model.SeqLevels,
		"bag":                model.Inventory.List(),
		"equipment":          model.Equipment.Export(),
		"equipment_bonus":    model.Equipment.TotalBonus(),
		"is_running":         isRunning,
		"seq_id":             model.CurrentSeqID,
		"seq_level":          model.CurrentSeqLevel(),
		"active_sub_project": model.ActiveSubProject,
	}
}
