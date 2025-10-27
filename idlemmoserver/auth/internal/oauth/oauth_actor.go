package oauth

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/nats-io/nats.go"
)

// OAuthActor OAuth处理Actor
type OAuthActor struct {
	providers map[string]OAuthProvider
}

// OAuthGetAuthURLRequest 获取授权URL请求
type OAuthGetAuthURLRequest struct {
	Platform string
	State    string
	ReplyTo  *actor.PID
}

// OAuthCallbackRequest OAuth回调请求
type OAuthCallbackRequest struct {
	Platform string
	Code     string
	State    string
	ReplyTo  *actor.PID
}

// OAuthGetUserInfoRequest 获取用户信息请求
type OAuthGetUserInfoRequest struct {
	Platform string
	Code     string
	ReplyTo  *actor.PID
}

// NewOAuthActor 创建OAuth Actor
func NewOAuthActor(providers map[string]OAuthProvider) actor.Producer {
	return func() actor.Actor {
		return &OAuthActor{
			providers: providers,
		}
	}
}

// Receive 处理消息
func (a *OAuthActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *OAuthGetAuthURLRequest:
		a.handleGetAuthURL(ctx, msg)
	case *OAuthCallbackRequest:
		a.handleOAuthCallback(ctx, msg)
	case *OAuthGetUserInfoRequest:
		a.handleGetUserInfo(ctx, msg)
	default:
		log.Printf("OAuthActor: unknown message type %T", msg)
	}
}

// handleGetAuthURL 处理获取授权URL
func (a *OAuthActor) handleGetAuthURL(ctx actor.Context, msg *OAuthGetAuthURLRequest) {
	provider, exists := a.providers[msg.Platform]
	if !exists {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported platform: %s", msg.Platform),
		})
		return
	}

	// 生成授权URL
	authURL := provider.GetAuthURL(msg.State)

	ctx.Respond(&OAuthResponse{
		Success: true,
		AuthURL: authURL,
	})
}

// handleOAuthCallback 处理OAuth回调
func (a *OAuthActor) handleOAuthCallback(ctx actor.Context, msg *OAuthCallbackRequest) {
	provider, exists := a.providers[msg.Platform]
	if !exists {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported platform: %s", msg.Platform),
		})
		return
	}

	// 用授权码换取访问令牌
	token, err := provider.ExchangeToken(msg.Code)
	if err != nil {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to exchange token: %v", err),
		})
		return
	}

	// 获取用户信息
	userInfo, err := provider.GetUserInfo(token)
	if err != nil {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get user info: %v", err),
		})
		return
	}

	ctx.Respond(&OAuthResponse{
		Success:  true,
		Token:    token,
		UserInfo: userInfo,
	})
}

// handleGetUserInfo 处理获取用户信息
func (a *OAuthActor) handleGetUserInfo(ctx actor.Context, msg *OAuthGetUserInfoRequest) {
	provider, exists := a.providers[msg.Platform]
	if !exists {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Unsupported platform: %s", msg.Platform),
		})
		return
	}

	// 用授权码换取访问令牌
	token, err := provider.ExchangeToken(msg.Code)
	if err != nil {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to exchange token: %v", err),
		})
		return
	}

	// 获取用户信息
	userInfo, err := provider.GetUserInfo(token)
	if err != nil {
		ctx.Respond(&OAuthResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get user info: %v", err),
		})
		return
	}

	ctx.Respond(&OAuthResponse{
		Success:  true,
		UserInfo: userInfo,
	})
}

// OAuthURLReplyActor 授权URL回复Actor
type OAuthURLReplyActor struct {
	msg *nats.Msg
}

func (a *OAuthURLReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *OAuthResponse:
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal OAuth URL response: %v", err)
			return
		}

		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to OAuth URL request: %v", err)
		}

		ctx.Stop(ctx.Self())
	}
}

// OAuthCallbackReplyActor OAuth回调回复Actor
type OAuthCallbackReplyActor struct {
	msg *nats.Msg
}

func (a *OAuthCallbackReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *OAuthResponse:
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal OAuth callback response: %v", err)
			return
		}

		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to OAuth callback request: %v", err)
		}

		ctx.Stop(ctx.Self())
	}
}

// OAuthUserInfoReplyActor 用户信息回复Actor
type OAuthUserInfoReplyActor struct {
	msg *nats.Msg
}

func (a *OAuthUserInfoReplyActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *OAuthResponse:
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal OAuth user info response: %v", err)
			return
		}

		if err := a.msg.Respond(data); err != nil {
			log.Printf("Failed to respond to OAuth user info request: %v", err)
		}

		ctx.Stop(ctx.Self())
	}
}
