package login

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
