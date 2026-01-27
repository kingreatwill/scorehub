package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	appauth "scorehub/internal/auth"
	appconfig "scorehub/internal/config"
	"scorehub/internal/store"
)

type AuthHandlers struct {
	cfg appconfig.Config
	st  *store.Store
}

func NewAuthHandlers(cfg appconfig.Config, st *store.Store) *AuthHandlers {
	return &AuthHandlers{cfg: cfg, st: st}
}

type devLoginRequest struct {
	OpenID    string `json:"openid"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func (h *AuthHandlers) DevLogin(ctx context.Context, c *app.RequestContext) {
	if !h.cfg.DevAuth {
		writeError(c, http.StatusForbidden, "forbidden", "dev auth disabled")
		return
	}

	var req devLoginRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	req.OpenID = strings.TrimSpace(req.OpenID)
	if req.OpenID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "openid required")
		return
	}

	u, err := h.st.UpsertUserByOpenID(ctx, req.OpenID, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	token, err := appauth.SignToken([]byte(h.cfg.TokenSecret), u.ID, 30*24*time.Hour)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "sign token failed", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":        u.ID,
			"openid":    u.WeChatOpenID,
			"nickname":  u.WeChatNickname,
			"avatarUrl": u.WeChatAvatarURL,
		},
	})
}

type wechatLoginRequest struct {
	Code      string `json:"code"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func (h *AuthHandlers) WechatLogin(ctx context.Context, c *app.RequestContext) {
	var req wechatLoginRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	openid, err := h.exchangeWeChatCode(ctx, strings.TrimSpace(req.Code))
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	u, err := h.st.UpsertUserByOpenID(ctx, openid, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	token, err := appauth.SignToken([]byte(h.cfg.TokenSecret), u.ID, 30*24*time.Hour)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "sign token failed", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]any{
			"id":        u.ID,
			"openid":    u.WeChatOpenID,
			"nickname":  u.WeChatNickname,
			"avatarUrl": u.WeChatAvatarURL,
		},
	})
}

func (h *AuthHandlers) exchangeWeChatCode(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", &appError{message: "code required"}
	}
	if h.cfg.WeChatAppID == "" || h.cfg.WeChatSecret == "" {
		if h.cfg.DevAuth {
			// 本地开发：允许直接把 code 当 openid 使用
			return code, nil
		}
		return "", &appError{message: "wechat appid/secret not configured"}
	}
	openid, err := exchangeCodeWithWeChatAPI(ctx, h.cfg.WeChatAppID, h.cfg.WeChatSecret, code)
	if err != nil {
		if h.cfg.DevAuth {
			return "", &appError{message: "wechat login failed: " + err.Error()}
		}
		return "", &appError{message: "wechat login failed"}
	}
	if strings.TrimSpace(openid) == "" {
		return "", &appError{message: "wechat openid empty"}
	}
	return openid, nil
}

type appError struct {
	message string
}

func (e *appError) Error() string { return e.message }
