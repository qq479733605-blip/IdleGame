package domain

import (
	"errors"
	"math/rand"
	"time"
)

// PlayerSnapshot 表示可用于加载或保存的玩家数据快照。
type PlayerSnapshot struct {
	SeqLevels         map[string]int
	Inventory         map[string]int64
	Exp               int64
	Equipment         map[string]EquipmentState
	OfflineLimitHours int64
}

// LoadOutcome 描述加载玩家数据后的结果（需发送给客户端的消息）。
type LoadOutcome struct {
	Messages []any
}

// OfflineRewardResult 表示离线奖励结算结果。
type OfflineRewardResult struct {
	Gains    int64
	Items    map[string]int64
	Duration time.Duration
}

// AttachConnResult 包含处理连接绑定后的信息。
type AttachConnResult struct {
	OfflineReward      *OfflineRewardResult
	ReconnectedPayload map[string]any
	InventoryErrors    []error
}

// SequenceResultData 封装序列执行结束时的结算数据。
type SequenceResultData struct {
	Gains        int64
	Rare         []string
	Items        []Item
	SeqID        string
	Level        int
	CurExp       int64
	Leveled      bool
	SubProjectID string
}

// SequenceResultOutcome 描述处理序列结算后的动作。
type SequenceResultOutcome struct {
	Messages        []any
	ShouldPersist   bool
	InventoryErrors []error
}

// PlayerDomain 封装围绕玩家模型的业务逻辑。
type PlayerDomain struct {
	model *PlayerModel
}

// NewPlayerDomain 构建一个新的玩家领域服务。
func NewPlayerDomain(playerID string) *PlayerDomain {
	return &PlayerDomain{model: NewPlayerModel(playerID)}
}

// Model 暴露底层模型供协调层读取。
func (d *PlayerDomain) Model() *PlayerModel {
	return d.model
}

// ApplySnapshot 根据持久层快照更新玩家模型并构建客户端响应。
func (d *PlayerDomain) ApplySnapshot(snapshot *PlayerSnapshot) *LoadOutcome {
	if snapshot == nil {
		d.model.SeqLevels = d.defaultSeqLevels()
		d.model.Inventory = NewInventory(200)
		d.model.Equipment = NewEquipmentLoadout()
		d.model.Exp = 0
		d.model.OfflineLimit = 10 * time.Hour
		return &LoadOutcome{Messages: []any{map[string]any{"type": "S_NewPlayer"}}}
	}

	d.model.SeqLevels = make(map[string]int)
	for id, level := range snapshot.SeqLevels {
		d.model.SeqLevels[id] = level
	}
	for seqID := range Sequences {
		if d.model.SeqLevels[seqID] <= 0 {
			d.model.SeqLevels[seqID] = 1
		}
	}

	d.model.Inventory = NewInventory(200)
	for id, count := range snapshot.Inventory {
		_ = d.model.Inventory.AddItem(Item{ID: id, Name: id}, count)
	}

	d.model.Equipment = NewEquipmentLoadout()
	if snapshot.Equipment != nil {
		d.model.Equipment.ImportState(snapshot.Equipment)
	}

	d.model.Exp = snapshot.Exp
	if snapshot.OfflineLimitHours > 0 {
		d.model.OfflineLimit = time.Duration(snapshot.OfflineLimitHours) * time.Hour
	}

	payload := map[string]any{
		"type":                "S_LoadOK",
		"exp":                 d.model.Exp,
		"bag":                 d.model.Inventory.List(),
		"offline_limit_hours": snapshot.OfflineLimitHours,
		"equipment":           d.model.Equipment.Export(),
		"equipment_bonus":     d.model.Equipment.TotalBonus(),
	}
	return &LoadOutcome{Messages: []any{payload}}
}

// AttachConnection 处理连接绑定及状态同步。
func (d *PlayerDomain) AttachConnection(requestState bool) *AttachConnResult {
	d.model.IsOnline = true
	d.model.LastActive = time.Now()
	result := &AttachConnResult{}

	if requestState {
		reward := d.calculateOfflineRewards()
		if reward != nil {
			d.model.Exp += reward.Gains
			reward.Items = ensureItemMap(reward.Items)
			for itemID, count := range reward.Items {
				if err := d.model.Inventory.AddItem(Item{ID: itemID, Name: itemID}, count); err != nil {
					result.InventoryErrors = append(result.InventoryErrors, err)
				}
			}
			result.OfflineReward = reward
		}
		result.ReconnectedPayload = d.buildReconnectedPayload()
	} else {
		result.ReconnectedPayload = map[string]any{"type": "S_Reconnected", "msg": "重连成功"}
	}

	d.model.OfflineStart = time.Time{}
	return result
}

func ensureItemMap(m map[string]int64) map[string]int64 {
	if m != nil {
		return m
	}
	return map[string]int64{}
}

func (d *PlayerDomain) calculateOfflineRewards() *OfflineRewardResult {
	if d.model.OfflineStart.IsZero() {
		return nil
	}
	duration := time.Since(d.model.OfflineStart)
	if duration <= 0 || duration >= d.model.OfflineLimit {
		return &OfflineRewardResult{Duration: duration, Items: map[string]int64{}}
	}

	gains := int64(0)
	items := make(map[string]int64)
	seconds := duration.Seconds()

	for seqID, level := range d.model.SeqLevels {
		if level <= 0 {
			continue
		}
		cfg, exists := GetSequenceConfig(seqID)
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

	if gains == 0 && len(items) == 0 {
		return nil
	}

	return &OfflineRewardResult{Gains: gains, Items: items, Duration: duration}
}

func (d *PlayerDomain) buildReconnectedPayload() map[string]any {
	currentLevel := 0
	if d.model.CurrentSeqID != "" {
		currentLevel = d.model.SeqLevels[d.model.CurrentSeqID]
	}
	return map[string]any{
		"type":               "S_Reconnected",
		"msg":                "重连成功",
		"seq_id":             d.model.CurrentSeqID,
		"seq_level":          currentLevel,
		"exp":                d.model.Exp,
		"bag":                d.model.Inventory.List(),
		"is_running":         d.model.CurrentSeqID != "",
		"seq_levels":         d.model.SeqLevels,
		"equipment":          d.model.Equipment.Export(),
		"equipment_bonus":    d.model.Equipment.TotalBonus(),
		"active_sub_project": d.model.ActiveSubProject,
	}
}

// BuildLoginPayload 构建登录成功返回的消息。
func (d *PlayerDomain) BuildLoginPayload() map[string]any {
	level := 0
	if d.model.CurrentSeqID != "" {
		level = d.model.SeqLevels[d.model.CurrentSeqID]
	}
	return map[string]any{
		"type":               "S_LoginOK",
		"msg":                "登录成功",
		"playerId":           d.model.ID,
		"exp":                d.model.Exp,
		"seq_levels":         d.model.SeqLevels,
		"bag":                d.model.Inventory.List(),
		"equipment":          d.model.Equipment.Export(),
		"equipment_bonus":    d.model.Equipment.TotalBonus(),
		"is_running":         d.model.CurrentSeqID != "",
		"seq_id":             d.model.CurrentSeqID,
		"seq_level":          level,
		"active_sub_project": d.model.ActiveSubProject,
	}
}

// MarkOffline 记录玩家离线的时间点。
func (d *PlayerDomain) MarkOffline(now time.Time) {
	d.model.IsOnline = false
	d.model.OfflineStart = now
}

// OnReconnect 处理玩家重连。
func (d *PlayerDomain) OnReconnect() map[string]any {
	d.model.IsOnline = true
	d.model.LastActive = time.Now()
	return map[string]any{"type": "S_ReconnectOK"}
}

// ShouldExpire 判断玩家是否超过离线限制。
func (d *PlayerDomain) ShouldExpire(now time.Time) bool {
	if d.model.IsOnline || d.model.OfflineStart.IsZero() {
		return false
	}
	return now.Sub(d.model.OfflineStart) > d.model.OfflineLimit
}

// ClearSequenceState 在序列结束时重置状态。
func (d *PlayerDomain) ClearSequenceState() {
	d.model.CurrentSeqID = ""
	d.model.ActiveSubProject = ""
}

// ExportSaveState 导出用于持久化的快照。
func (d *PlayerDomain) ExportSaveState() *PlayerSnapshot {
	return &PlayerSnapshot{
		SeqLevels:         copySeqLevels(d.model.SeqLevels),
		Inventory:         d.model.Inventory.List(),
		Exp:               d.model.Exp,
		Equipment:         d.model.Equipment.ExportState(),
		OfflineLimitHours: int64(d.model.OfflineLimit / time.Hour),
	}
}

func copySeqLevels(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ApplySequenceResult 应用序列结算结果并生成客户端响应。
func (d *PlayerDomain) ApplySequenceResult(data *SequenceResultData) (*SequenceResultOutcome, error) {
	if data == nil {
		return nil, errors.New("sequence result is nil")
	}
	outcome := &SequenceResultOutcome{ShouldPersist: true}
	for _, it := range data.Items {
		if err := d.model.Inventory.AddItem(it, 1); err != nil {
			outcome.InventoryErrors = append(outcome.InventoryErrors, err)
		}
	}
	if data.SeqID != "" {
		d.model.SeqLevels[data.SeqID] = data.Level
	}
	d.model.ActiveSubProject = data.SubProjectID
	d.model.Exp += data.Gains
	d.model.LastActive = time.Now()

	payload := map[string]any{
		"type":            "S_SeqResult",
		"gains":           data.Gains,
		"rare":            data.Rare,
		"bag":             d.model.Inventory.List(),
		"seq_id":          data.SeqID,
		"level":           data.Level,
		"cur_exp":         data.CurExp,
		"leveled":         data.Leveled,
		"items":           data.Items,
		"sub_project_id":  data.SubProjectID,
		"equipment_bonus": d.model.Equipment.TotalBonus(),
	}
	outcome.Messages = []any{payload}
	return outcome, nil
}

// GetEquipmentBonus 返回当前的装备加成。
func (d *PlayerDomain) GetEquipmentBonus() EquipmentBonus {
	return d.model.Equipment.TotalBonus()
}

// HasActiveSequence 判断是否有正在运行的序列。
func (d *PlayerDomain) HasActiveSequence() bool {
	return d.model.CurrentSeqID != ""
}

// GetSequenceLevel 获取指定序列的等级。
func (d *PlayerDomain) GetSequenceLevel(seqID string) int {
	return d.model.SeqLevels[seqID]
}

func (d *PlayerDomain) defaultSeqLevels() map[string]int {
	levels := make(map[string]int)
	for seqID := range Sequences {
		levels[seqID] = 1
	}
	return levels
}
