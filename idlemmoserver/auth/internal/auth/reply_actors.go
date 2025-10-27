package auth

import (
	"encoding/json"
	"log"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// AuthReplyActor 认证回复Actor
type AuthReplyActor struct {
	msg *nats.Msg
}

// NewAuthReplyActor 创建认证回复Actor
func NewAuthReplyActor(msg *nats.Msg) actor.Actor {
	return &AuthReplyActor{msg: msg}
}

// Receive 处理回复消息
func (a *AuthReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgAuthenticateUserResult:
		// 序列化回复
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal auth result: %v", err)
			return
		}

		// 发送回复
		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to auth request: %v", err)
		}

		// 停止自己
		ctx.Stop(ctx.Self())
	}
}

// RegisterReplyActor 注册回复Actor
type RegisterReplyActor struct {
	msg *nats.Msg
}

// NewRegisterReplyActor 创建注册回复Actor
func NewRegisterReplyActor(msg *nats.Msg) actor.Actor {
	return &RegisterReplyActor{msg: msg}
}

// Receive 处理回复消息
func (a *RegisterReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgRegisterUserResult:
		// 序列化回复
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal register result: %v", err)
			return
		}

		// 发送回复
		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to register request: %v", err)
		}

		// 停止自己
		ctx.Stop(ctx.Self())
	}
}

// GetUserReplyActor 获取用户回复Actor
type GetUserReplyActor struct {
	msg *nats.Msg
}

// NewGetUserReplyActor 创建获取用户回复Actor
func NewGetUserReplyActor(msg *nats.Msg) actor.Actor {
	return &GetUserReplyActor{msg: msg}
}

// Receive 处理回复消息
func (a *GetUserReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgGetUserByPlayerIDResult:
		// 序列化回复
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal get user result: %v", err)
			return
		}

		// 发送回复
		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to get user request: %v", err)
		}

		// 停止自己
		ctx.Stop(ctx.Self())
	}
}

// ValidateTokenReplyActor Token验证回复Actor
type ValidateTokenReplyActor struct {
	msg *nats.Msg
}

// NewValidateTokenReplyActor 创建Token验证回复Actor
func NewValidateTokenReplyActor(msg *nats.Msg) actor.Actor {
	return &ValidateTokenReplyActor{msg: msg}
}

// Receive 处理回复消息
func (a *ValidateTokenReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgValidateTokenResult:
		// 序列化回复
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal validate token result: %v", err)
			return
		}

		// 发送回复
		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to validate token request: %v", err)
		}

		// 停止自己
		ctx.Stop(ctx.Self())
	}
}

// GetPlayerByTokenReplyActor 根据Token获取PlayerID回复Actor
type GetPlayerByTokenReplyActor struct {
	msg *nats.Msg
}

// NewGetPlayerByTokenReplyActor 创建根据Token获取PlayerID回复Actor
func NewGetPlayerByTokenReplyActor(msg *nats.Msg) actor.Actor {
	return &GetPlayerByTokenReplyActor{msg: msg}
}

// Receive 处理回复消息
func (a *GetPlayerByTokenReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgGetPlayerByTokenResult:
		// 序列化回复
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal get player by token result: %v", err)
			return
		}

		// 发送回复
		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to get player by token request: %v", err)
		}

		// 停止自己
		ctx.Stop(ctx.Self())
	}
}
