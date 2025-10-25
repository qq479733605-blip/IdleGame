package actors

import (
	"fmt"

	gamedomain "idlemmoserver/internal/domain"

	"github.com/asynkron/protoactor-go/actor"
)

// LoginCommand 处理客户端的登录请求。
type LoginCommand struct {
	Type string `json:"type"`
}

// Execute 登录指令只需将当前状态返回给客户端。
func (c *LoginCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	payload := p.domain.BuildLoginPayload(p.model, p.currentSeq != nil)
	p.sendToClient(payload)
	return nil
}

// StartSequenceCommand 请求启动指定的修炼序列。
type StartSequenceCommand struct {
	Type         string `json:"type"`
	SeqID        string `json:"seq_id"`
	Target       int64  `json:"target"`
	SubProjectID string `json:"sub_project_id"`
}

// Execute 负责触发 PlayerActor 内部的序列启动流程。
func (c *StartSequenceCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	if c.SeqID == "" {
		return fmt.Errorf("缺少修炼序列 ID")
	}
	return p.startSequence(ctx, c.SeqID, c.SubProjectID)
}

// StopSequenceCommand 请求停止当前修炼。
type StopSequenceCommand struct {
	Type string `json:"type"`
}

// Execute 停止当前运行的修炼序列。
func (c *StopSequenceCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	p.stopCurrentSequence(ctx, true)
	return nil
}

// ListBagCommand 查询背包信息。
type ListBagCommand struct {
	Type string `json:"type"`
}

// Execute 返回当前背包列表。
func (c *ListBagCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	p.sendToClient(map[string]any{"type": "S_BagInfo", "bag": p.model.Inventory.List()})
	return nil
}

// ListEquipmentCommand 查询装备状态。
type ListEquipmentCommand struct {
	Type string `json:"type"`
}

// Execute 返回当前装备信息（不包含目录）。
func (c *ListEquipmentCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	p.sendEquipmentState(false)
	return nil
}

// EquipItemCommand 处理装备穿戴。
type EquipItemCommand struct {
	Type        string `json:"type"`
	ItemID      string `json:"item_id"`
	Enhancement int    `json:"enhancement"`
}

// Execute 尝试穿戴装备并推送最新状态。
func (c *EquipItemCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	if c.ItemID == "" {
		return fmt.Errorf("请选择要装备的物品")
	}
	if err := p.domain.EquipItem(p.model, c.ItemID, c.Enhancement); err != nil {
		return err
	}
	p.pushEquipmentBonus(ctx)
	p.sendEquipmentChanged()
	return nil
}

// UnequipItemCommand 处理卸下装备。
type UnequipItemCommand struct {
	Type string `json:"type"`
	Slot string `json:"slot"`
}

// Execute 卸下指定槽位的装备并推送状态。
func (c *UnequipItemCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	if c.Slot == "" {
		return fmt.Errorf("请选择要卸下的位置")
	}
	slot := gamedomain.EquipmentSlot(c.Slot)
	if err := p.domain.UnequipItem(p.model, slot); err != nil {
		return err
	}
	p.pushEquipmentBonus(ctx)
	p.sendEquipmentChanged()
	return nil
}

// UseItemCommand 使用背包物品。
type UseItemCommand struct {
	Type   string `json:"type"`
	ItemID string `json:"item_id"`
	Count  int64  `json:"count"`
}

// Execute 移除物品并按照既定规则增加经验。
func (c *UseItemCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	if c.ItemID == "" {
		return fmt.Errorf("缺少物品 ID")
	}
	gain, err := p.domain.UseItem(p.model, c.ItemID, c.Count)
	if err != nil {
		return err
	}
	p.sendToClient(map[string]any{
		"type":    "S_ItemUsed",
		"item_id": c.ItemID,
		"count":   c.Count,
		"effect":  "exp+10",
		"exp":     p.model.Exp,
		"gain":    gain,
	})
	return nil
}

// RemoveItemCommand 主动丢弃背包物品。
type RemoveItemCommand struct {
	Type   string `json:"type"`
	ItemID string `json:"item_id"`
	Count  int64  `json:"count"`
}

// Execute 删除指定数量的物品。
func (c *RemoveItemCommand) Execute(ctx actor.Context, p *PlayerActor) error {
	if c.ItemID == "" {
		return fmt.Errorf("缺少物品 ID")
	}
	if err := p.domain.RemoveItem(p.model, c.ItemID, c.Count); err != nil {
		return err
	}
	p.sendToClient(map[string]any{"type": "S_ItemRemoved", "item_id": c.ItemID, "count": c.Count})
	return nil
}
