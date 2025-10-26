package domain

// Formula 接口预留，用于未来技能修炼、炼丹、装备强化等特殊序列
type Formula interface {
	CalcGains(level int) int64
	RollItems(level int) []Item
	MaybeTriggerEvent(level int) *RareEvent
}
