package oauth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/idle-server/common"
	"github.com/nats-io/nats.go"
)

// OAuthService OAuth服务
type OAuthService struct {
	nc        *nats.Conn
	system    *actor.ActorSystem
	oauthPID  *actor.PID
	providers map[string]OAuthProvider
}

// NewOAuthService 创建OAuth服务
func NewOAuthService() *OAuthService {
	service := &OAuthService{
		providers: make(map[string]OAuthProvider),
	}

	// 注册OAuth提供者
	service.registerProviders()

	return service
}

// registerProviders 注册OAuth提供者
func (s *OAuthService) registerProviders() {
	// 注册微信提供者
	wechatProvider := NewWeChatProvider()
	s.providers["wechat"] = wechatProvider

	// 注册QQ提供者
	qqProvider := NewQQProvider()
	s.providers["qq"] = qqProvider

	// 未来可以添加更多平台
	// douyinProvider := NewDouyinProvider()
	// s.providers["douyin"] = douyinProvider
}

// Start 启动OAuth服务
func (s *OAuthService) Start(nc *nats.Conn, system *actor.ActorSystem) error {
	s.nc = nc
	s.system = system

	// 创建OAuth Actor
	props := actor.PropsFromProducer(NewOAuthActor(s.providers))
	s.oauthPID = system.Root.Spawn(props)

	// 注册NATS处理器
	if err := s.registerNATSHandlers(s.oauthPID); err != nil {
		return fmt.Errorf("failed to register NATS handlers: %w", err)
	}

	log.Printf("OAuth service started successfully")
	return nil
}

// registerNATSHandlers 注册NATS处理器
func (s *OAuthService) registerNATSHandlers(oauthPID *actor.PID) error {
	// 获取授权URL处理器
	authURLSub, err := s.nc.Subscribe(common.OAuthAuthURLSubject, func(msg *nats.Msg) {
		s.handleGetAuthURL(oauthPID, msg)
	})
	if err != nil {
		return err
	}

	// OAuth回调处理器
	callbackSub, err := s.nc.Subscribe(common.OAuthCallbackSubject, func(msg *nats.Msg) {
		s.handleOAuthCallback(oauthPID, msg)
	})
	if err != nil {
		return err
	}

	// 获取用户信息处理器
	userInfoSub, err := s.nc.Subscribe(common.OAuthUserInfoSubject, func(msg *nats.Msg) {
		s.handleGetUserInfo(oauthPID, msg)
	})
	if err != nil {
		return err
	}

	// 使用变量避免编译错误
	_ = authURLSub
	_ = callbackSub
	_ = userInfoSub

	log.Printf("OAuth NATS handlers registered")
	return nil
}

// handleGetAuthURL 处理获取授权URL请求
func (s *OAuthService) handleGetAuthURL(oauthPID *actor.PID, msg *nats.Msg) {
	var req OAuthRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal get auth URL request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &OAuthURLReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	// 发送请求到OAuth Actor
	oauthReq := &OAuthGetAuthURLRequest{
		Platform: req.Platform,
		State:    req.State,
		ReplyTo:  replyPID,
	}

	s.system.Root.Send(oauthPID, oauthReq)
}

// handleOAuthCallback 处理OAuth回调
func (s *OAuthService) handleOAuthCallback(oauthPID *actor.PID, msg *nats.Msg) {
	var req OAuthRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal OAuth callback request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &OAuthCallbackReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	// 发送请求到OAuth Actor
	oauthReq := &OAuthCallbackRequest{
		Platform: req.Platform,
		Code:     req.Code,
		State:    req.State,
		ReplyTo:  replyPID,
	}

	s.system.Root.Send(oauthPID, oauthReq)
}

// handleGetUserInfo 处理获取用户信息请求
func (s *OAuthService) handleGetUserInfo(oauthPID *actor.PID, msg *nats.Msg) {
	var req OAuthRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		log.Printf("Failed to unmarshal get user info request: %v", err)
		return
	}

	// 创建回复Actor
	replyProps := actor.PropsFromProducer(func() actor.Actor {
		return &OAuthUserInfoReplyActor{msg: msg}
	})
	replyPID := s.system.Root.Spawn(replyProps)

	// 发送请求到OAuth Actor
	oauthReq := &OAuthGetUserInfoRequest{
		Platform: req.Platform,
		Code:     req.Code,
		ReplyTo:  replyPID,
	}

	s.system.Root.Send(oauthPID, oauthReq)
}

// GenerateState 生成OAuth状态参数
func GenerateState() string {
	return fmt.Sprintf("oauth_%d", time.Now().UnixNano())
}

// ParseState 解析OAuth状态参数
func ParseState(state string) (timestamp int64, ok bool) {
	var ts int64
	if _, err := fmt.Sscanf(state, "oauth_%d", &ts); err != nil {
		return 0, false
	}
	return ts, true
}

// IsStateValid 检查状态是否有效（防止CSRF攻击）
func IsStateValid(state string, maxAge time.Duration) bool {
	timestamp, ok := ParseState(state)
	if !ok {
		return false
	}

	// 检查时间戳是否在有效期内
	return time.Since(time.Unix(timestamp, 0)) < maxAge
}

// BuildAuthURL 构建授权URL
func BuildAuthURL(baseURL string, params map[string]string) string {
	u, _ := url.Parse(baseURL)
	q := u.Query()

	for key, value := range params {
		q.Set(key, value)
	}

	u.RawQuery = q.Encode()
	return u.String()
}
