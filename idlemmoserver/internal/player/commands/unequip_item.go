package player

import (
	"encoding/json"

	"idlemmoserver/internal/common"
)

type unequipRequest struct {
	Slot string `json:"slot"`
}

type UnequipItemCommand struct {
	req unequipRequest
}

func NewUnequipItemCommand(raw []byte) Command {
	var req unequipRequest
	_ = json.Unmarshal(raw, &req)
	return &UnequipItemCommand{req: req}
}

func (c *UnequipItemCommand) Execute(ctx *CommandContext) error {
	slot := common.EquipmentSlot(c.req.Slot)
	item := ctx.Domain.Unequip(slot)
	if item == nil {
		ctx.Player.sendError("slot empty")
		return nil
	}
	ctx.Domain.RestoreEquippedItem(common.ItemDrop{ID: item.Definition.ID, Name: item.Definition.Name})
	ctx.Player.pushEquipmentBonus(ctx.ActorCtx)
	return nil
}
