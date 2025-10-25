package service

import (
	"errors"

	"idlemmoserver/internal/domain"
)

// ItemService encapsulates item usage and inventory management rules.
type ItemService struct {
	model *domain.PlayerModel
}

func NewItemService(model *domain.PlayerModel) *ItemService {
	return &ItemService{model: model}
}

func (s *ItemService) UseItem(itemID string, count int64) (map[string]any, error) {
	if count <= 0 {
		return nil, errors.New("invalid count")
	}
	if err := s.model.Inventory.RemoveItem(itemID, count); err != nil {
		return nil, err
	}

	s.model.Exp += count * 10
	return map[string]any{
		"type":    "S_ItemUsed",
		"item_id": itemID,
		"count":   count,
		"effect":  "exp+10",
		"exp":     s.model.Exp,
	}, nil
}

func (s *ItemService) RemoveItem(itemID string, count int64) error {
	return s.model.Inventory.RemoveItem(itemID, count)
}
