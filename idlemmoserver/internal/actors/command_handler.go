package actors

import (
	"encoding/json"
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

// Command 定义客户端指令的统一执行接口。
type Command interface {
	Execute(ctx actor.Context, p *PlayerActor) error
}

type commandFactory func() Command

// CommandHandler 负责注册与分发客户端指令，实现 Actor 与命令之间的解耦。
type CommandHandler struct {
	registry map[string]commandFactory
}

// NewCommandHandler 构建一个默认注册常用指令的处理器。
func NewCommandHandler() *CommandHandler {
	handler := &CommandHandler{registry: make(map[string]commandFactory)}
	handler.Register("C_Login", func() Command { return &LoginCommand{} })
	handler.Register("C_StartSeq", func() Command { return &StartSequenceCommand{} })
	handler.Register("C_StopSeq", func() Command { return &StopSequenceCommand{} })
	handler.Register("C_ListBag", func() Command { return &ListBagCommand{} })
	handler.Register("C_ListEquipment", func() Command { return &ListEquipmentCommand{} })
	handler.Register("C_EquipItem", func() Command { return &EquipItemCommand{} })
	handler.Register("C_UnequipItem", func() Command { return &UnequipItemCommand{} })
	handler.Register("C_UseItem", func() Command { return &UseItemCommand{} })
	handler.Register("C_RemoveItem", func() Command { return &RemoveItemCommand{} })
	return handler
}

// Register 用于在运行时动态注册新的命令类型。
func (h *CommandHandler) Register(name string, factory commandFactory) {
	h.registry[name] = factory
}

// Handle 根据客户端发送的原始数据解析出指令并执行。
func (h *CommandHandler) Handle(ctx actor.Context, p *PlayerActor, payload *MsgClientPayload) error {
	var base baseMsg
	if err := json.Unmarshal(payload.Raw, &base); err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	factory, ok := h.registry[base.Type]
	if !ok {
		return fmt.Errorf("未知指令: %s", base.Type)
	}
	cmd := factory()
	if err := json.Unmarshal(payload.Raw, cmd); err != nil {
		return fmt.Errorf("解析指令数据失败: %w", err)
	}
	if attachable, ok := cmd.(interface{ AttachPayload(*MsgClientPayload) }); ok {
		attachable.AttachPayload(payload)
	}
	return cmd.Execute(ctx, p)
}
