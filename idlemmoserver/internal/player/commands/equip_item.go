package player

import (
	"encoding/json"

	"idlemmoserver/internal/common"
)

type equipItemRequest struct {
	ItemID      string `json:"item_id"`
	Enhancement int    `json:"enhancement"`
}

type EquipItemCommand struct {
	req equipItemRequest
}

func NewEquipItemCommand(raw []byte) Command {
	var req equipItemRequest
	_ = json.Unmarshal(raw, &req)
	return &EquipItemCommand{req: req}
}

func (c *EquipItemCommand) Execute(ctx *CommandContext) error {
	if c.req.ItemID == "" {
		ctx.Player.sendError("请选择要装备的物品")
		return ErrItemNotEquippable
	}
	replaced, err := ctx.Domain.EquipItem(c.req.ItemID, c.req.Enhancement)
	if err != nil {
		ctx.Player.sendError(err.Error())
		return err
	}
	if replaced != nil {
		ctx.Domain.RestoreEquippedItem(common.ItemDrop{ID: replaced.Definition.ID, Name: replaced.Definition.Name})
	}
	ctx.Player.pushEquipmentBonus(ctx.ActorCtx)
	return nil
}
