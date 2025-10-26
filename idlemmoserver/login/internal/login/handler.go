package login

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
)

// LoginActor 登录处理Actor
type LoginActor struct {
	userRepo common.UserRepository
}

// NewLoginActor 创建新的登录Actor
func NewLoginActor(userRepo common.UserRepository) actor.Producer {
	return func() actor.Actor {
		return &LoginActor{
			userRepo: userRepo,
		}
	}
}

// Receive 处理消息
func (a *LoginActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgAuthenticateUser:
		a.handleAuthenticateUser(ctx, msg)
	case *common.MsgRegisterUser:
		a.handleRegisterUser(ctx, msg)
	case *common.MsgGetUserByPlayerID:
		a.handleGetUserByPlayerID(ctx, msg)
	default:
		log.Printf("LoginActor: unknown message type %T", msg)
	}
}

// handleAuthenticateUser 处理用户认证
func (a *LoginActor) handleAuthenticateUser(ctx actor.Context, msg *common.MsgAuthenticateUser) {
	// 获取用户数据
	user, err := a.userRepo.GetUser(msg.Username)
	if err != nil {
		ctx.Respond(&common.MsgAuthenticateUserResult{
			Success: false,
			Message: "用户不存在",
		})
		return
	}

	// 验证密码
	if !verifyPassword(msg.Password, user.Password) {
		ctx.Respond(&common.MsgAuthenticateUserResult{
			Success: false,
			Message: "密码错误",
		})
		return
	}

	// 更新最后登录时间
	a.userRepo.UpdateLastLogin(msg.Username)

	// 生成Token
	token := GenerateToken()

	ctx.Respond(&common.MsgAuthenticateUserResult{
		Success:  true,
		Message:  "登录成功",
		PlayerID: user.PlayerID,
		Token:    token,
	})
}

// handleRegisterUser 处理用户注册
func (a *LoginActor) handleRegisterUser(ctx actor.Context, msg *common.MsgRegisterUser) {
	// 检查用户是否已存在
	if a.userRepo.UserExists(msg.Username) {
		ctx.Respond(&common.MsgRegisterUserResult{
			Success: false,
			Message: "用户名已存在",
		})
		return
	}

	// 生成PlayerID
	playerID := generatePlayerID()

	// 创建用户数据
	user := &common.UserData{
		Username:  msg.Username,
		Password:  hashPassword(msg.Password),
		PlayerID:  playerID,
		CreatedAt: time.Now(),
		LastLogin: time.Now(),
	}

	// 保存用户
	if err := a.userRepo.SaveUser(user); err != nil {
		ctx.Respond(&common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: " + err.Error(),
		})
		return
	}

	ctx.Respond(&common.MsgRegisterUserResult{
		Success:  true,
		Message:  "注册成功",
		PlayerID: playerID,
	})
}

// handleGetUserByPlayerID 根据PlayerID获取用户
func (a *LoginActor) handleGetUserByPlayerID(ctx actor.Context, msg *common.MsgGetUserByPlayerID) {
	user, err := a.userRepo.GetUserByPlayerID(msg.PlayerID)
	if err != nil {
		ctx.Respond(&common.MsgGetUserByPlayerIDResult{
			User:   nil,
			Exists: false,
		})
		return
	}

	ctx.Respond(&common.MsgGetUserByPlayerIDResult{
		User:   user,
		Exists: true,
	})
}

// hashPassword 对密码进行哈希
func hashPassword(password string) string {
	// 临时使用简单哈希，生产环境应使用bcrypt
	return password
}

// verifyPassword 验证密码
func verifyPassword(password, hash string) bool {
	return hashPassword(password) == hash
}

// generatePlayerID 生成PlayerID
func generatePlayerID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
