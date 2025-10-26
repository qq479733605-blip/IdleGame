package gateway

import (
	"time"

	"idlemmoserver/internal/common"
	"idlemmoserver/internal/logx"

	"github.com/asynkron/protoactor-go/actor"
)

type AuthActor struct {
	repo common.UserRepository
}

func NewAuthActor(repo common.UserRepository) actor.Actor {
	return &AuthActor{repo: repo}
}

func (a *AuthActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgRegisterUser:
		a.handleRegister(ctx, msg)
	case *common.MsgAuthenticateUser:
		a.handleAuthenticate(ctx, msg)
	case *common.MsgGetUserByPlayerID:
		a.handleGetByPlayer(ctx, msg)
	}
}

func (a *AuthActor) handleRegister(ctx actor.Context, msg *common.MsgRegisterUser) {
	if a.repo.UserExists(msg.Username) {
		ctx.Send(msg.ReplyTo, &common.MsgRegisterUserResult{Success: false, Message: "用户名已存在"})
		return
	}
	user := &common.UserData{
		Username:  msg.Username,
		Password:  common.HashPassword(msg.Password),
		PlayerID:  common.GeneratePlayerID(),
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}
	if err := a.repo.SaveUser(user); err != nil {
		logx.Error("保存用户失败", "username", msg.Username, "error", err)
		ctx.Send(msg.ReplyTo, &common.MsgRegisterUserResult{Success: false, Message: "注册失败，请稍后重试"})
		return
	}
	ctx.Send(msg.ReplyTo, &common.MsgRegisterUserResult{Success: true, Message: "注册成功", PlayerID: user.PlayerID})
}

func (a *AuthActor) handleAuthenticate(ctx actor.Context, msg *common.MsgAuthenticateUser) {
	user, err := a.repo.GetUser(msg.Username)
	if err != nil {
		ctx.Send(msg.ReplyTo, &common.MsgAuthenticateUserResult{Success: false, Message: "用户名或密码错误"})
		return
	}
	if !common.VerifyPassword(msg.Password, user.Password) {
		logx.Warn("用户登录失败", "username", msg.Username, "reason", "wrong password")
		ctx.Send(msg.ReplyTo, &common.MsgAuthenticateUserResult{Success: false, Message: "用户名或密码错误"})
		return
	}
	if err := a.repo.UpdateLastLogin(msg.Username); err != nil {
		logx.Warn("更新最后登录时间失败", "username", msg.Username, "error", err)
	}
	ctx.Send(msg.ReplyTo, &common.MsgAuthenticateUserResult{Success: true, Message: "登录成功", PlayerID: user.PlayerID})
}

func (a *AuthActor) handleGetByPlayer(ctx actor.Context, msg *common.MsgGetUserByPlayerID) {
	user, err := a.repo.GetUserByPlayerID(msg.PlayerID)
	if err != nil {
		ctx.Send(msg.ReplyTo, &common.MsgGetUserByPlayerIDResult{User: nil, Exists: false})
		return
	}
	ctx.Send(msg.ReplyTo, &common.MsgGetUserByPlayerIDResult{User: user, Exists: true})
}
