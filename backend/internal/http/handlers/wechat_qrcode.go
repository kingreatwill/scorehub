package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type wechatAccessTokenCache struct {
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

var globalWeChatTokenCache wechatAccessTokenCache

type wechatAccessTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

func getWeChatAccessToken(ctx context.Context, appid, secret string) (string, error) {
	globalWeChatTokenCache.mu.Lock()
	defer globalWeChatTokenCache.mu.Unlock()

	// 预留 60 秒安全窗口，避免边界失效
	if globalWeChatTokenCache.token != "" && time.Now().Before(globalWeChatTokenCache.expiresAt.Add(-60*time.Second)) {
		return globalWeChatTokenCache.token, nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.weixin.qq.com",
		Path:   "/cgi-bin/token",
	}
	q := u.Query()
	q.Set("grant_type", "client_credential")
	q.Set("appid", appid)
	q.Set("secret", secret)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var r wechatAccessTokenResp
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if r.ErrCode != 0 {
		return "", fmt.Errorf("wechat err %d: %s", r.ErrCode, r.ErrMsg)
	}
	if r.AccessToken == "" || r.ExpiresIn <= 0 {
		return "", fmt.Errorf("wechat invalid token response")
	}

	globalWeChatTokenCache.token = r.AccessToken
	globalWeChatTokenCache.expiresAt = time.Now().Add(time.Duration(r.ExpiresIn) * time.Second)
	return r.AccessToken, nil
}

type wechatGetWxaCodeReq struct {
	Scene string `json:"scene"`
	Page  string `json:"page,omitempty"`
	Width int    `json:"width,omitempty"`
}

type wechatErrResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type wechatAPIError struct {
	Code int
	Msg  string
}

func (e *wechatAPIError) Error() string {
	return fmt.Sprintf("wechat err %d: %s", e.Code, e.Msg)
}

func getWeChatMiniProgramCode(ctx context.Context, accessToken string, scene string, page string) ([]byte, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.weixin.qq.com",
		Path:   "/wxa/getwxacodeunlimit",
	}
	q := u.Query()
	q.Set("access_token", accessToken)
	u.RawQuery = q.Encode()

	payload := wechatGetWxaCodeReq{
		Scene: scene,
		Page:  page,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(b)) > 0 && bytes.TrimSpace(b)[0] == '{' {
		var er wechatErrResp
		if json.Unmarshal(b, &er) == nil && er.ErrCode != 0 {
			return nil, &wechatAPIError{Code: er.ErrCode, Msg: er.ErrMsg}
		}
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("wechat empty qrcode response")
	}
	return b, nil
}
