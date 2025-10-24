package domain

type Item struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DropChance float64 `json:"drop_chance"`
	Value      int64   `json:"value"`
}

type RareEvent struct {
	Name     string  `json:"name"`
	Effect   string  `json:"effect"`
	MultGain float64 `json:"mult_gain"`
}
