package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/golang-jwt/jwt/v5"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// AuthActor 认证处理Actor
type AuthActor struct {
	jwtSecret []byte
	nc        *nats.Conn
}

// NewAuthActor 创建新的认证Actor
func NewAuthActor(jwtSecret []byte, nc *nats.Conn) actor.Producer {
	return func() actor.Actor {
		return &AuthActor{
			jwtSecret: jwtSecret,
			nc:        nc,
		}
	}
}

// Receive 处理消息
func (a *AuthActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *common.MsgAuthenticateUser:
		a.handleAuthenticateUser(ctx, msg)
	case *common.MsgRegisterUser:
		a.handleRegisterUser(ctx, msg)
	case *common.MsgGetUserByPlayerID:
		a.handleGetUserByPlayerID(ctx, msg)
	case *common.MsgValidateToken:
		a.handleValidateToken(ctx, msg)
	case *common.MsgGetPlayerByToken:
		a.handleGetPlayerByToken(ctx, msg)
	default:
		log.Printf("AuthActor: unknown message type %T", msg)
	}
}

// handleAuthenticateUser 处理用户认证
func (a *AuthActor) handleAuthenticateUser(ctx actor.Context, msg *common.MsgAuthenticateUser) {
	// 这个方法现在应该通过NATS处理，但由于我们已经改在service层直接处理
	// 这个方法实际上不会被调用到，保留结构以备将来扩展
	log.Printf("handleAuthenticateUser called, but processing is now done in service layer")
	ctx.Respond(&common.MsgAuthenticateUserResult{
		Success: false,
		Message: "认证处理已移至服务层",
	})
}

// handleRegisterUser 处理用户注册
func (a *AuthActor) handleRegisterUser(ctx actor.Context, msg *common.MsgRegisterUser) {
	// 通过NATS请求检查用户是否存在
	existsMsg := &common.MsgUserExists{
		Username: msg.Username,
	}

	data, err := json.Marshal(existsMsg)
	if err != nil {
		log.Printf("Failed to marshal user exists message: %v", err)
		ctx.Respond(&common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 序列化请求出错",
		})
		return
	}

	// 发送NATS请求到Persist服务
	resp, err := a.nc.Request(common.PersistUserExistsSubject, data, 5*time.Second)
	if err != nil {
		log.Printf("Failed to check user existence via NATS: %v", err)
		ctx.Respond(&common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 无法连接到持久化服务",
		})
		return
	}

	var existsResult common.MsgUserExistsResult
	if err := json.Unmarshal(resp.Data, &existsResult); err != nil {
		log.Printf("Failed to unmarshal user exists result: %v", err)
		ctx.Respond(&common.MsgRegisterUserResult{
			Success: false,
			Message: "注册失败: 解析响应出错",
		})
		return
	}

	// 如果用户已存在，返回错误
	if existsResult.Exists {
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

	// 通过NATS保存用户数据
	if err := a.saveUser(ctx, user); err != nil {
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
func (a *AuthActor) handleGetUserByPlayerID(ctx actor.Context, msg *common.MsgGetUserByPlayerID) {
	// 这个方法现在应该通过NATS处理，但由于我们已经改在service层直接处理
	// 这个方法实际上不会被调用到，保留结构以备将来扩展
	log.Printf("handleGetUserByPlayerID called, but processing is now done in service layer")
	ctx.Respond(&common.MsgGetUserByPlayerIDResult{
		User:   nil,
		Exists: false,
	})
}

// handleValidateToken 处理Token验证
func (a *AuthActor) handleValidateToken(ctx actor.Context, msg *common.MsgValidateToken) {
	claims, err := a.validateJWT(msg.Token)
	if err != nil {
		ctx.Respond(&common.MsgValidateTokenResult{
			Valid:   false,
			Message: err.Error(),
		})
		return
	}

	playerID, ok := (*claims)["playerID"].(string)
	if !ok {
		ctx.Respond(&common.MsgValidateTokenResult{
			Valid:   false,
			Message: "invalid player ID in token",
		})
		return
	}

	ctx.Respond(&common.MsgValidateTokenResult{
		Valid:    true,
		PlayerID: playerID,
	})
}

// handleGetPlayerByToken 处理根据Token获取PlayerID
func (a *AuthActor) handleGetPlayerByToken(ctx actor.Context, msg *common.MsgGetPlayerByToken) {
	claims, err := a.validateJWT(msg.Token)
	if err != nil {
		ctx.Respond(&common.MsgGetPlayerByTokenResult{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	playerID, ok := (*claims)["playerID"].(string)
	if !ok {
		ctx.Respond(&common.MsgGetPlayerByTokenResult{
			Success: false,
			Message: "invalid player ID in token",
		})
		return
	}

	ctx.Respond(&common.MsgGetPlayerByTokenResult{
		Success:  true,
		PlayerID: playerID,
	})
}

// generateJWT 生成JWT Token
func (a *AuthActor) generateJWT(playerID string) (string, error) {
	claims := jwt.MapClaims{
		"playerID": playerID,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// validateJWT 验证JWT Token
func (a *AuthActor) validateJWT(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, fmt.Errorf("invalid token")
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

// ========== NATS通信辅助方法 ==========

// checkUserExists 通过NATS检查用户是否存在
func (a *AuthActor) checkUserExists(ctx actor.Context, username string) error {
	existsMsg := &common.MsgUserExists{
		Username: username,
	}

	data, err := json.Marshal(existsMsg)
	if err != nil {
		return err
	}

	resp, err := a.nc.Request(common.PersistUserExistsSubject, data, 5*time.Second)
	if err != nil {
		return err
	}

	var existsResult common.MsgUserExistsResult
	if err := json.Unmarshal(resp.Data, &existsResult); err != nil {
		return err
	}

	// 如果用户存在，返回错误
	if existsResult.Exists {
		return fmt.Errorf("user already exists")
	}

	return nil
}

// saveUser 通过NATS保存用户数据
func (a *AuthActor) saveUser(ctx actor.Context, user *common.UserData) error {
	saveMsg := &common.MsgSaveUser{
		UserData: user,
	}

	data, err := json.Marshal(saveMsg)
	if err != nil {
		return err
	}

	// 发送NATS请求到Persist服务
	resp, err := a.nc.Request(common.PersistSaveUserSubject, data, 5*time.Second)
	if err != nil {
		return err
	}

	// Persist服务不返回数据，只需要检查是否有错误
	if resp != nil {
		log.Printf("User save response received: %s", string(resp.Data))
	}

	return nil
}
