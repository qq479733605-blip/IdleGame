package domain

type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DropChance  float64 `json:"drop_chance"`
	Value       int64   `json:"value"`
	IsEquipment bool    `json:"is_equipment"` // 标记是否为装备
}

type RareEvent struct {
	Name     string  `json:"name"`
	Effect   string  `json:"effect"`
	MultGain float64 `json:"mult_gain"`
}
