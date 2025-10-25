package service

import (
	"errors"

	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"
)

// EquipmentService encapsulates equipment loadout management.
type EquipmentService struct {
	model *domain.PlayerModel
}

func NewEquipmentService(model *domain.PlayerModel) *EquipmentService {
	return &EquipmentService{model: model}
}

func (s *EquipmentService) EquipItem(itemID string, enhancement int) error {
	if itemID == "" {
		return errors.New("请选择要装备的物品")
	}
	def, ok := domain.GetEquipmentDefinition(itemID)
	if !ok {
		return errors.New("该物品无法装备")
	}
	if err := s.model.Inventory.RemoveItem(itemID, 1); err != nil {
		return err
	}

	replaced := s.model.Equipment.Equip(def, enhancement)
	if replaced != nil {
		if err := s.model.Inventory.AddItem(domain.Item{ID: replaced.Definition.ID, Name: replaced.Definition.Name}, 1); err != nil {
			s.model.Equipment.Equip(replaced.Definition, replaced.Enhancement)
			_ = s.model.Inventory.AddItem(domain.Item{ID: def.ID, Name: def.Name}, 1)
			return errors.New("背包空间不足")
		}
	}

	logx.Info("equip item", "player", s.model.PlayerID, "item", def.ID)
	return nil
}

func (s *EquipmentService) UnequipItem(slot domain.EquipmentSlot) error {
	item := s.model.Equipment.Unequip(slot)
	if item == nil {
		return errors.New("该位置没有装备")
	}
	if err := s.model.Inventory.AddItem(domain.Item{ID: item.Definition.ID, Name: item.Definition.Name}, 1); err != nil {
		s.model.Equipment.Equip(item.Definition, item.Enhancement)
		return errors.New("背包空间不足")
	}

	logx.Info("unequip item", "player", s.model.PlayerID, "slot", slot)
	return nil
}

func (s *EquipmentService) EquipmentState(includeCatalog bool) map[string]any {
	payload := map[string]any{
		"type":      "S_EquipmentState",
		"equipment": s.model.Equipment.Export(),
		"bonus":     s.model.Equipment.TotalBonus(),
		"bag":       s.model.Inventory.List(),
	}
	if includeCatalog {
		payload["catalog"] = domain.GetEquipmentCatalogSummary()
	}
	return payload
}

func (s *EquipmentService) EquipmentChangedPayload() map[string]any {
	return map[string]any{
		"type":      "S_EquipmentChanged",
		"equipment": s.model.Equipment.Export(),
		"bonus":     s.model.Equipment.TotalBonus(),
		"bag":       s.model.Inventory.List(),
	}
}
