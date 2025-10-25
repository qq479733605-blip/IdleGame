package domain

import (
	"fmt"
	"time"
)

// CommandError 表示可传递给客户端的业务错误。
type CommandError struct {
	Message string
}

func (e *CommandError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// PlayerCommand 定义所有玩家指令应实现的接口。
type PlayerCommand interface {
	Execute(domain *PlayerDomain) (*CommandResult, error)
}

// CommandResult 描述执行指令后产生的副作用。
type CommandResult struct {
	Responses          []any
	StartPlan          *SequenceStartPlan
	StopCurrent        bool
	EquipmentBonus     EquipmentBonus
	PushEquipmentBonus bool
	Persist            bool
}

// SequenceStartPlan 描述启动序列所需的上下文。
type SequenceStartPlan struct {
	SeqID          string
	Level          int
	SubProject     *SequenceSubProject
	EquipmentBonus EquipmentBonus
}

// CommandHandler 负责调度执行指令。
type CommandHandler struct {
	domain *PlayerDomain
}

// NewPlayerCommandHandler 创建一个命令调度器。
func NewPlayerCommandHandler(domain *PlayerDomain) *CommandHandler {
	return &CommandHandler{domain: domain}
}

// Handle 执行指令。
func (h *CommandHandler) Handle(cmd PlayerCommand) (*CommandResult, error) {
	if cmd == nil {
		return nil, &CommandError{Message: "未知指令"}
	}
	return cmd.Execute(h.domain)
}

// LoginCommand 处理登录成功后的状态同步。
type LoginCommand struct{}

func NewLoginCommand() *LoginCommand { return &LoginCommand{} }

func (c *LoginCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	payload := domain.BuildLoginPayload()
	return &CommandResult{Responses: []any{payload}}, nil
}

// StartSequenceCommand 请求启动新的序列。
type StartSequenceCommand struct {
	SeqID        string
	SubProjectID string
}

func NewStartSequenceCommand(seqID, subProjectID string) *StartSequenceCommand {
	return &StartSequenceCommand{SeqID: seqID, SubProjectID: subProjectID}
}

func (c *StartSequenceCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	if c.SeqID == "" {
		return nil, &CommandError{Message: "请选择要运行的序列"}
	}
	cfg, exists := GetSequenceConfig(c.SeqID)
	if !exists {
		return nil, &CommandError{Message: "sequence not found"}
	}
	level := domain.GetSequenceLevel(c.SeqID)
	if level <= 0 {
		level = 1
	}

	var subProject *SequenceSubProject
	if c.SubProjectID != "" {
		sp, ok := cfg.GetSubProject(c.SubProjectID)
		if !ok {
			return nil, &CommandError{Message: "sub project not found"}
		}
		if level < sp.UnlockLevel {
			return nil, &CommandError{Message: "子项目未解锁"}
		}
		subProject = sp
	}

	domain.model.CurrentSeqID = c.SeqID
	if subProject != nil {
		domain.model.ActiveSubProject = subProject.ID
	} else {
		domain.model.ActiveSubProject = ""
	}
	domain.model.LastActive = time.Now()

	bonus := domain.GetEquipmentBonus()
	payload := map[string]any{
		"type":            "S_SeqStarted",
		"seq_id":          c.SeqID,
		"level":           level,
		"sub_project_id":  domain.model.ActiveSubProject,
		"tick_interval":   cfg.EffectiveInterval(subProject).Seconds(),
		"equipment_bonus": bonus,
	}

	return &CommandResult{
		Responses:   []any{payload},
		StartPlan:   &SequenceStartPlan{SeqID: c.SeqID, Level: level, SubProject: subProject, EquipmentBonus: bonus},
		StopCurrent: true,
	}, nil
}

// StopSequenceCommand 请求停止当前序列。
type StopSequenceCommand struct{}

func NewStopSequenceCommand() *StopSequenceCommand { return &StopSequenceCommand{} }

func (c *StopSequenceCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	if !domain.HasActiveSequence() {
		return &CommandResult{}, nil
	}
	domain.ClearSequenceState()
	domain.model.LastActive = time.Now()

	payload := map[string]any{
		"type":               "S_SeqEnded",
		"is_running":         false,
		"seq_id":             "",
		"seq_level":          0,
		"active_sub_project": "",
	}
	return &CommandResult{
		Responses:   []any{payload},
		StopCurrent: true,
	}, nil
}

// ListBagCommand 返回背包信息。
type ListBagCommand struct{}

func NewListBagCommand() *ListBagCommand { return &ListBagCommand{} }

func (c *ListBagCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	payload := map[string]any{"type": "S_BagInfo", "bag": domain.model.Inventory.List()}
	domain.model.LastActive = time.Now()
	return &CommandResult{Responses: []any{payload}}, nil
}

// ListEquipmentCommand 返回装备状态。
type ListEquipmentCommand struct {
	IncludeCatalog bool
}

func NewListEquipmentCommand(includeCatalog bool) *ListEquipmentCommand {
	return &ListEquipmentCommand{IncludeCatalog: includeCatalog}
}

func (c *ListEquipmentCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	payload := map[string]any{
		"type":      "S_EquipmentState",
		"equipment": domain.model.Equipment.Export(),
		"bonus":     domain.model.Equipment.TotalBonus(),
		"bag":       domain.model.Inventory.List(),
	}
	if c.IncludeCatalog {
		payload["catalog"] = GetEquipmentCatalogSummary()
	}
	domain.model.LastActive = time.Now()
	return &CommandResult{Responses: []any{payload}}, nil
}

// EquipItemCommand 处理装备穿戴。
type EquipItemCommand struct {
	ItemID      string
	Enhancement int
}

func NewEquipItemCommand(itemID string, enhancement int) *EquipItemCommand {
	return &EquipItemCommand{ItemID: itemID, Enhancement: enhancement}
}

func (c *EquipItemCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	if c.ItemID == "" {
		return nil, &CommandError{Message: "请选择要装备的物品"}
	}
	def, ok := GetEquipmentDefinition(c.ItemID)
	if !ok {
		return nil, &CommandError{Message: "该物品无法装备"}
	}
	if err := domain.model.Inventory.RemoveItem(c.ItemID, 1); err != nil {
		return nil, &CommandError{Message: err.Error()}
	}

	replaced := domain.model.Equipment.Equip(def, c.Enhancement)
	if replaced != nil {
		if err := domain.model.Inventory.AddItem(Item{ID: replaced.Definition.ID, Name: replaced.Definition.Name}, 1); err != nil {
			domain.model.Equipment.Equip(replaced.Definition, replaced.Enhancement)
			_ = domain.model.Inventory.AddItem(Item{ID: def.ID, Name: def.Name}, 1)
			return nil, &CommandError{Message: "背包空间不足"}
		}
	}

	domain.model.LastActive = time.Now()
	payload := map[string]any{
		"type":      "S_EquipmentChanged",
		"equipment": domain.model.Equipment.Export(),
		"bonus":     domain.model.Equipment.TotalBonus(),
		"bag":       domain.model.Inventory.List(),
	}
	bonus := domain.model.Equipment.TotalBonus()
	return &CommandResult{
		Responses:          []any{payload},
		PushEquipmentBonus: true,
		EquipmentBonus:     bonus,
	}, nil
}

// UnequipItemCommand 处理卸下装备。
type UnequipItemCommand struct {
	Slot EquipmentSlot
}

func NewUnequipItemCommand(slot EquipmentSlot) *UnequipItemCommand {
	return &UnequipItemCommand{Slot: slot}
}

func (c *UnequipItemCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	if c.Slot == "" {
		return nil, &CommandError{Message: "请选择要卸下的位置"}
	}
	item := domain.model.Equipment.Unequip(c.Slot)
	if item == nil {
		return nil, &CommandError{Message: "该位置没有装备"}
	}
	if err := domain.model.Inventory.AddItem(Item{ID: item.Definition.ID, Name: item.Definition.Name}, 1); err != nil {
		domain.model.Equipment.Equip(item.Definition, item.Enhancement)
		return nil, &CommandError{Message: "背包空间不足"}
	}

	domain.model.LastActive = time.Now()
	payload := map[string]any{
		"type":      "S_EquipmentChanged",
		"equipment": domain.model.Equipment.Export(),
		"bonus":     domain.model.Equipment.TotalBonus(),
		"bag":       domain.model.Inventory.List(),
	}
	bonus := domain.model.Equipment.TotalBonus()
	return &CommandResult{
		Responses:          []any{payload},
		PushEquipmentBonus: true,
		EquipmentBonus:     bonus,
	}, nil
}

// UseItemCommand 处理使用背包物品。
type UseItemCommand struct {
	ItemID string
	Count  int64
}

func NewUseItemCommand(itemID string, count int64) *UseItemCommand {
	return &UseItemCommand{ItemID: itemID, Count: count}
}

func (c *UseItemCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	if c.Count <= 0 {
		return nil, &CommandError{Message: "invalid count"}
	}
	if err := domain.model.Inventory.RemoveItem(c.ItemID, c.Count); err != nil {
		return nil, &CommandError{Message: err.Error()}
	}
	domain.model.Exp += c.Count * 10
	domain.model.LastActive = time.Now()

	payload := map[string]any{
		"type":    "S_ItemUsed",
		"item_id": c.ItemID,
		"count":   c.Count,
		"effect":  "exp+10",
		"exp":     domain.model.Exp,
	}
	return &CommandResult{Responses: []any{payload}, Persist: true}, nil
}

// RemoveItemCommand 处理移除物品。
type RemoveItemCommand struct {
	ItemID string
	Count  int64
}

func NewRemoveItemCommand(itemID string, count int64) *RemoveItemCommand {
	return &RemoveItemCommand{ItemID: itemID, Count: count}
}

func (c *RemoveItemCommand) Execute(domain *PlayerDomain) (*CommandResult, error) {
	if err := domain.model.Inventory.RemoveItem(c.ItemID, c.Count); err != nil {
		return nil, &CommandError{Message: err.Error()}
	}
	domain.model.LastActive = time.Now()
	payload := map[string]any{"type": "S_ItemRemoved", "item_id": c.ItemID, "count": c.Count}
	return &CommandResult{Responses: []any{payload}, Persist: true}, nil
}

// NewCommandError 创建一个业务错误。
func NewCommandError(msg string) error {
	if msg == "" {
		return fmt.Errorf("command error")
	}
	return &CommandError{Message: msg}
}
