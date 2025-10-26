package login

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
)

// HTTPAuthReplyActor HTTP认证回复Actor
type HTTPAuthReplyActor struct {
	responseChan chan interface{}
}

func (a *HTTPAuthReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgAuthenticateUserResult:
		select {
		case a.responseChan <- msg:
		default:
		}
		ctx.Stop(ctx.Self())
	}
}

// HTTPRegisterReplyActor HTTP注册回复Actor
type HTTPRegisterReplyActor struct {
	responseChan chan interface{}
}

func (a *HTTPRegisterReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgRegisterUserResult:
		select {
		case a.responseChan <- msg:
		default:
		}
		ctx.Stop(ctx.Self())
	}
}
