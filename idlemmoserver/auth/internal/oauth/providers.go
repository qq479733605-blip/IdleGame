package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// WeChatProvider 微信OAuth提供者
type WeChatProvider struct {
	config OAuthConfig
}

// NewWeChatProvider 创建微信OAuth提供者
func NewWeChatProvider() OAuthProvider {
	return &WeChatProvider{
		config: WeChatOAuthConfig,
	}
}

// GetAuthURL 获取微信授权URL
func (p *WeChatProvider) GetAuthURL(state string) string {
	params := map[string]string{
		"appid":         p.config.ClientID,
		"redirect_uri":  p.config.RedirectURL,
		"response_type": "code",
		"scope":         "snsapi_login",
		"state":         state,
	}

	return BuildAuthURL(p.config.AuthURL, params)
}

// ExchangeToken 用授权码换取访问令牌
func (p *WeChatProvider) ExchangeToken(code string) (*OAuthToken, error) {
	params := url.Values{}
	params.Set("appid", p.config.ClientID)
	params.Set("secret", p.config.ClientSecret)
	params.Set("code", code)
	params.Set("grant_type", "authorization_code")

	resp, err := http.Get(p.config.TokenURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		OpenID       string `json:"openid"`
		Scope        string `json:"scope"`
		UnionID      string `json:"unionid"`
		ErrCode      int    `json:"errcode"`
		ErrMsg       string `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	if tokenResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat API error: %d - %s", tokenResp.ErrCode, tokenResp.ErrMsg)
	}

	return &OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenResp.ExpiresIn,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
		OpenID:       tokenResp.OpenID, // 添加OpenID字段
	}, nil
}

// GetUserInfo 获取微信用户信息
func (p *WeChatProvider) GetUserInfo(token *OAuthToken) (*OAuthUserInfo, error) {
	params := url.Values{}
	params.Set("access_token", token.AccessToken)
	params.Set("openid", token.OpenID)
	params.Set("lang", "zh_CN")

	resp, err := http.Get(p.config.UserInfoURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	var userResp struct {
		OpenID     string   `json:"openid"`
		Nickname   string   `json:"nickname"`
		Sex        int      `json:"sex"`
		Province   string   `json:"province"`
		City       string   `json:"city"`
		Country    string   `json:"country"`
		HeadImgURL string   `json:"headimgurl"`
		Privilege  []string `json:"privilege"`
		UnionID    string   `json:"unionid"`
		ErrCode    int      `json:"errcode"`
		ErrMsg     string   `json:"errmsg"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %v", err)
	}

	if userResp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat API error: %d - %s", userResp.ErrCode, userResp.ErrMsg)
	}

	return &OAuthUserInfo{
		ID:       userResp.OpenID,
		Username: userResp.OpenID,
		Nickname: userResp.Nickname,
		Avatar:   userResp.HeadImgURL,
		Platform: "wechat",
		Raw: map[string]interface{}{
			"openid":     userResp.OpenID,
			"nickname":   userResp.Nickname,
			"sex":        userResp.Sex,
			"province":   userResp.Province,
			"city":       userResp.City,
			"country":    userResp.Country,
			"headimgurl": userResp.HeadImgURL,
			"privilege":  userResp.Privilege,
			"unionid":    userResp.UnionID,
		},
	}, nil
}

// GetPlatform 获取平台名称
func (p *WeChatProvider) GetPlatform() string {
	return "wechat"
}

// QQProvider QQ OAuth提供者
type QQProvider struct {
	config OAuthConfig
}

// NewQQProvider 创建QQ OAuth提供者
func NewQQProvider() OAuthProvider {
	return &QQProvider{
		config: QQOAuthConfig,
	}
}

// GetAuthURL 获取QQ授权URL
func (p *QQProvider) GetAuthURL(state string) string {
	params := map[string]string{
		"response_type": "code",
		"client_id":     p.config.ClientID,
		"redirect_uri":  p.config.RedirectURL,
		"scope":         "get_user_info",
		"state":         state,
	}

	return BuildAuthURL(p.config.AuthURL, params)
}

// ExchangeToken 用授权码换取访问令牌
func (p *QQProvider) ExchangeToken(code string) (*OAuthToken, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("client_id", p.config.ClientID)
	params.Set("client_secret", p.config.ClientSecret)
	params.Set("code", code)
	params.Set("redirect_uri", p.config.RedirectURL)

	resp, err := http.Get(p.config.TokenURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		ExpiresIn    int64  `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("empty access token received")
	}

	return &OAuthToken{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    tokenResp.ExpiresIn,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// GetUserInfo 获取QQ用户信息
func (p *QQProvider) GetUserInfo(token *OAuthToken) (*OAuthUserInfo, error) {
	// 首先获取OpenID
	openID, err := p.getOpenID(token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get OpenID: %v", err)
	}

	// 获取用户信息
	params := url.Values{}
	params.Set("access_token", token.AccessToken)
	params.Set("oauth_consumer_key", p.config.ClientID)
	params.Set("openid", openID)

	resp, err := http.Get(p.config.UserInfoURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}
	defer resp.Body.Close()

	var userResp struct {
		Ret        int    `json:"ret"`
		Msg        string `json:"msg"`
		Nickname   string `json:"nickname"`
		Gender     string `json:"gender"`
		Province   string `json:"province"`
		City       string `json:"city"`
		FigureURL  string `json:"figureurl"`
		FigureURL1 string `json:"figureurl_1"`
		FigureURL2 string `json:"figureurl_2"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %v", err)
	}

	if userResp.Ret != 0 {
		return nil, fmt.Errorf("QQ API error: %d - %s", userResp.Ret, userResp.Msg)
	}

	return &OAuthUserInfo{
		ID:       openID,
		Username: openID,
		Nickname: userResp.Nickname,
		Avatar:   userResp.FigureURL,
		Platform: "qq",
		Raw: map[string]interface{}{
			"openid":      openID,
			"nickname":    userResp.Nickname,
			"gender":      userResp.Gender,
			"province":    userResp.Province,
			"city":        userResp.City,
			"figureurl":   userResp.FigureURL,
			"figureurl_1": userResp.FigureURL1,
			"figureurl_2": userResp.FigureURL2,
		},
	}, nil
}

// getOpenID 获取QQ OpenID
func (p *QQProvider) getOpenID(accessToken string) (string, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)

	resp, err := http.Get("https://graph.qq.com/oauth2.0/me?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to get OpenID: %v", err)
	}
	defer resp.Body.Close()

	// QQ返回的是JSONP格式: callback( {"client_id":"...","openid":"..."} );
	var response struct {
		ClientID string `json:"client_id"`
		OpenID   string `json:"openid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode OpenID response: %v", err)
	}

	if response.OpenID == "" {
		return "", fmt.Errorf("empty OpenID received")
	}

	return response.OpenID, nil
}

// GetPlatform 获取平台名称
func (p *QQProvider) GetPlatform() string {
	return "qq"
}
