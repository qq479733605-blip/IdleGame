package actors

import (
	"idlemmoserver/internal/domain"
	"idlemmoserver/internal/logx"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// AuthActor 用户认证 Actor
type AuthActor struct {
	userRepo domain.UserRepository
}

// NewAuthActor 创建认证 Actor
func NewAuthActor(userRepo domain.UserRepository) actor.Actor {
	return &AuthActor{
		userRepo: userRepo,
	}
}

func (a *AuthActor) Receive(ctx actor.Context) {
	switch m := ctx.Message().(type) {
	case *MsgRegisterUser:
		a.handleRegisterUser(ctx, m)

	case *MsgAuthenticateUser:
		a.handleAuthenticateUser(ctx, m)

	case *MsgGetUserByPlayerID:
		a.handleGetUserByPlayerID(ctx, m)
	}
}

// handleRegisterUser 处理用户注册
func (a *AuthActor) handleRegisterUser(ctx actor.Context, m *MsgRegisterUser) {
	// 检查用户名是否已存在
	if a.userRepo.UserExists(m.Username) {
		ctx.Send(m.ReplyTo, &MsgRegisterUserResult{
			Success: false,
			Message: "用户名已存在",
		})
		return
	}

	// 创建新用户
	playerID := domain.GeneratePlayerID()
	user := &domain.UserData{
		Username:  m.Username,
		Password:  domain.HashPassword(m.Password),
		PlayerID:  playerID,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}

	// 保存用户数据
	if err := a.userRepo.SaveUser(user); err != nil {
		logx.Error("保存用户失败", "username", m.Username, "error", err)
		ctx.Send(m.ReplyTo, &MsgRegisterUserResult{
			Success: false,
			Message: "注册失败，请稍后重试",
		})
		return
	}

	logx.Info("用户注册成功", "username", m.Username, "playerID", playerID)
	ctx.Send(m.ReplyTo, &MsgRegisterUserResult{
		Success:  true,
		Message:  "注册成功",
		PlayerID: playerID,
	})
}

// handleAuthenticateUser 处理用户认证
func (a *AuthActor) handleAuthenticateUser(ctx actor.Context, m *MsgAuthenticateUser) {
	// 获取用户数据
	user, err := a.userRepo.GetUser(m.Username)
	if err != nil {
		ctx.Send(m.ReplyTo, &MsgAuthenticateUserResult{
			Success: false,
			Message: "用户名或密码错误",
		})
		return
	}

	// 验证密码
	if !domain.VerifyPassword(m.Password, user.Password) {
		logx.Warn("用户登录失败", "username", m.Username, "reason", "wrong password")
		ctx.Send(m.ReplyTo, &MsgAuthenticateUserResult{
			Success: false,
			Message: "用户名或密码错误",
		})
		return
	}

	// 更新最后登录时间
	if err := a.userRepo.UpdateLastLogin(m.Username); err != nil {
		logx.Warn("更新最后登录时间失败", "username", m.Username, "error", err)
		// 不影响登录流程
	}

	logx.Info("用户登录成功", "username", m.Username, "playerID", user.PlayerID)
	ctx.Send(m.ReplyTo, &MsgAuthenticateUserResult{
		Success:  true,
		Message:  "登录成功",
		PlayerID: user.PlayerID,
	})
}

// handleGetUserByPlayerID 处理通过 PlayerID 查找用户
func (a *AuthActor) handleGetUserByPlayerID(ctx actor.Context, m *MsgGetUserByPlayerID) {
	user, err := a.userRepo.GetUserByPlayerID(m.PlayerID)
	if err != nil {
		ctx.Send(m.ReplyTo, &MsgGetUserByPlayerIDResult{
			User:   nil,
			Exists: false,
		})
		return
	}

	ctx.Send(m.ReplyTo, &MsgGetUserByPlayerIDResult{
		User:   user,
		Exists: true,
	})
}
