package oauth

import (
	"net/url"
	"time"
)

// OAuthConfig OAuth配置
type OAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	UserInfoURL  string   `json:"user_info_url"`
	Scopes       []string `json:"scopes"`
}

// OAuthToken OAuth访问令牌
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	OpenID       string    `json:"openid,omitempty"`
}

// OAuthUserInfo OAuth用户信息
type OAuthUserInfo struct {
	ID       string                 `json:"id"`
	Username string                 `json:"username"`
	Nickname string                 `json:"nickname"`
	Avatar   string                 `json:"avatar"`
	Email    string                 `json:"email"`
	Platform string                 `json:"platform"`
	Raw      map[string]interface{} `json:"raw"`
}

// OAuthRequest OAuth请求
type OAuthRequest struct {
	Platform string `json:"platform"` // wechat, qq, douyin, etc.
	State    string `json:"state"`    // OAuth状态参数
	Code     string `json:"code"`     // 授权码
}

// OAuthResponse OAuth响应
type OAuthResponse struct {
	Success  bool           `json:"success"`
	Message  string         `json:"message"`
	Token    *OAuthToken    `json:"token,omitempty"`
	UserInfo *OAuthUserInfo `json:"user_info,omitempty"`
	AuthURL  string         `json:"auth_url,omitempty"`
}

// OAuthProvider OAuth提供者接口
type OAuthProvider interface {
	// GetAuthURL 获取授权URL
	GetAuthURL(state string) string

	// ExchangeToken 用授权码换取访问令牌
	ExchangeToken(code string) (*OAuthToken, error)

	// GetUserInfo 获取用户信息
	GetUserInfo(token *OAuthToken) (*OAuthUserInfo, error)

	// GetPlatform 获取平台名称
	GetPlatform() string
}

// 微信OAuth配置示例
var WeChatOAuthConfig = OAuthConfig{
	ClientID:     "your_wechat_appid",
	ClientSecret: "your_wechat_secret",
	RedirectURL:  "http://localhost:8002/auth/wechat/callback",
	AuthURL:      "https://open.weixin.qq.com/connect/qrconnect",
	TokenURL:     "https://api.weixin.qq.com/sns/oauth2/access_token",
	UserInfoURL:  "https://api.weixin.qq.com/sns/userinfo",
	Scopes:       []string{"snsapi_login"},
}

// QQ OAuth配置示例
var QQOAuthConfig = OAuthConfig{
	ClientID:     "your_qq_appid",
	ClientSecret: "your_qq_secret",
	RedirectURL:  "http://localhost:8002/auth/qq/callback",
	AuthURL:      "https://graph.qq.com/oauth2.0/authorize",
	TokenURL:     "https://graph.qq.com/oauth2.0/token",
	UserInfoURL:  "https://graph.qq.com/user/get_user_info",
	Scopes:       []string{"get_user_info"},
}
