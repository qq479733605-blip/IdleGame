package player

import "encoding/json"

type useItemRequest struct {
	ItemID string `json:"item_id"`
	Count  int64  `json:"count"`
}

type UseItemCommand struct {
	req useItemRequest
}

func NewUseItemCommand(raw []byte) Command {
	var req useItemRequest
	_ = json.Unmarshal(raw, &req)
	return &UseItemCommand{req: req}
}

func (c *UseItemCommand) Execute(ctx *CommandContext) error {
	gain, err := ctx.Domain.UseItem(c.req.ItemID, c.req.Count)
	if err != nil {
		ctx.Player.sendError(err.Error())
		return err
	}
	ctx.Player.sendToClient(map[string]any{
		"type":    "S_ItemUsed",
		"item_id": c.req.ItemID,
		"count":   c.req.Count,
		"effect":  "exp+10",
		"exp":     ctx.Domain.State().Exp,
		"gain":    gain,
	})
	return nil
}
