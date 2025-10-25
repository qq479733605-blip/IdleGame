package actors

import (
	"time"

	"idlemmoserver/internal/domain"
)

// PlayerModel 用于存储玩家的所有领域状态，作为 PlayerActor 与领域逻辑之间的共享数据结构。
type PlayerModel struct {
	ID               string
	SeqLevels        map[string]int
	Inventory        *domain.Inventory
	Equipment        *domain.EquipmentLoadout
	Exp              int64
	ActiveSubProject string
	CurrentSeqID     string
	IsOnline         bool
	OfflineStart     time.Time
	OfflineLimit     time.Duration
	LastActive       time.Time
}

// NewPlayerModel 创建默认的玩家领域模型，并初始化背包、装备与默认的离线时长限制。
func NewPlayerModel(id string) *PlayerModel {
	return &PlayerModel{
		ID:           id,
		SeqLevels:    make(map[string]int),
		Inventory:    domain.NewInventory(200),
		Equipment:    domain.NewEquipmentLoadout(),
		OfflineLimit: 10 * time.Hour,
		IsOnline:     true,
	}
}

// MarkOnline 将玩家状态更新为在线，同时刷新最后活动时间。
func (m *PlayerModel) MarkOnline(now time.Time) {
	m.IsOnline = true
	m.LastActive = now
}

// MarkOffline 将玩家状态标记为离线，并记录离线开始时间。
func (m *PlayerModel) MarkOffline(now time.Time) {
	m.IsOnline = false
	if m.OfflineStart.IsZero() {
		m.OfflineStart = now
	}
}

// ResetOffline 清除离线计时，用于玩家重新上线后重置离线收益结算。
func (m *PlayerModel) ResetOffline() {
	m.OfflineStart = time.Time{}
}

// CurrentSeqLevel 返回当前运行序列的等级，若未运行则为 0。
func (m *PlayerModel) CurrentSeqLevel() int {
	if m.CurrentSeqID == "" {
		return 0
	}
	return m.SeqLevels[m.CurrentSeqID]
}

// OfflineLimitHours 以小时为单位返回离线时长上限。
func (m *PlayerModel) OfflineLimitHours() int64 {
	if m.OfflineLimit <= 0 {
		return 0
	}
	return int64(m.OfflineLimit / time.Hour)
}

// SetOfflineLimitHours 设置离线时长上限（单位小时）。
func (m *PlayerModel) SetOfflineLimitHours(hours int64) {
	if hours <= 0 {
		return
	}
	m.OfflineLimit = time.Duration(hours) * time.Hour
}

// SetSequenceRunning 标记当前运行的序列信息。
func (m *PlayerModel) SetSequenceRunning(seqID string, subProjectID string) {
	m.CurrentSeqID = seqID
	m.ActiveSubProject = subProjectID
}

// ClearRunningSequence 清除当前运行的序列信息。
func (m *PlayerModel) ClearRunningSequence() {
	m.CurrentSeqID = ""
	m.ActiveSubProject = ""
}
