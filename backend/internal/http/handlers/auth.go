package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/jackc/pgx/v5/pgconn"

	appauth "scorehub/internal/auth"
	appconfig "scorehub/internal/config"
	"scorehub/internal/http/middleware"
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
		if isSchemaOutdated(err) {
			writeError(c, http.StatusInternalServerError, "internal", "db schema outdated: apply backend/sql/migrations/0004_auth.sql", err)
			return
		}
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
			"id":          u.ID,
			"openid":      valueOrEmpty(u.WeChatOpenID),
			"nickname":    u.WeChatNickname,
			"avatarUrl":   u.WeChatAvatarURL,
			"username":    valueOrEmpty(u.Username),
			"hasPassword": u.PasswordHash != nil && strings.TrimSpace(*u.PasswordHash) != "",
			"wechatBound": u.WeChatOpenID != nil && strings.TrimSpace(*u.WeChatOpenID) != "",
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
		if isSchemaOutdated(err) {
			writeError(c, http.StatusInternalServerError, "internal", "db schema outdated: apply backend/sql/migrations/0004_auth.sql", err)
			return
		}
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
			"id":          u.ID,
			"openid":      valueOrEmpty(u.WeChatOpenID),
			"nickname":    u.WeChatNickname,
			"avatarUrl":   u.WeChatAvatarURL,
			"username":    valueOrEmpty(u.Username),
			"hasPassword": u.PasswordHash != nil && strings.TrimSpace(*u.PasswordHash) != "",
			"wechatBound": u.WeChatOpenID != nil && strings.TrimSpace(*u.WeChatOpenID) != "",
		},
	})
}

type passwordLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandlers) PasswordLogin(ctx context.Context, c *app.RequestContext) {
	var req passwordLoginRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	username := normalizeUsername(req.Username)
	password := strings.TrimSpace(req.Password)
	if err := validateUsername(username); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := validatePassword(password); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	u, err := h.st.GetUserByUsername(ctx, username)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusUnauthorized, "unauthorized", "invalid username or password")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	if u.PasswordHash == nil || !appauth.VerifyPassword(*u.PasswordHash, password) {
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid username or password")
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
			"id":          u.ID,
			"openid":      valueOrEmpty(u.WeChatOpenID),
			"nickname":    u.WeChatNickname,
			"avatarUrl":   u.WeChatAvatarURL,
			"username":    valueOrEmpty(u.Username),
			"hasPassword": u.PasswordHash != nil && strings.TrimSpace(*u.PasswordHash) != "",
			"wechatBound": u.WeChatOpenID != nil && strings.TrimSpace(*u.WeChatOpenID) != "",
		},
	})
}

type setCredentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SetCredentials allows a logged-in user (typically WeChat login) to set username/password for the first time.
func (h *AuthHandlers) SetCredentials(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req setCredentialsRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	username := normalizeUsername(req.Username)
	password := strings.TrimSpace(req.Password)
	if err := validateUsername(username); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := validatePassword(password); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	hash, err := appauth.HashPassword(password)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "hash password failed", err)
		return
	}

	u, err := h.st.SetUserCredentials(ctx, uid, username, hash, time.Now())
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		if err == store.ErrConflict {
			writeError(c, http.StatusConflict, "conflict", "credentials already set or username taken")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":          u.ID,
			"openid":      valueOrEmpty(u.WeChatOpenID),
			"nickname":    u.WeChatNickname,
			"avatarUrl":   u.WeChatAvatarURL,
			"username":    valueOrEmpty(u.Username),
			"hasPassword": u.PasswordHash != nil && strings.TrimSpace(*u.PasswordHash) != "",
			"wechatBound": u.WeChatOpenID != nil && strings.TrimSpace(*u.WeChatOpenID) != "",
		},
	})
}

type changePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (h *AuthHandlers) ChangePassword(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req changePasswordRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	oldPwd := strings.TrimSpace(req.OldPassword)
	newPwd := strings.TrimSpace(req.NewPassword)
	if err := validatePassword(newPwd); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	u, err := h.st.GetUserByID(ctx, uid)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	if u.PasswordHash == nil || strings.TrimSpace(*u.PasswordHash) == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "password not set")
		return
	}
	if !appauth.VerifyPassword(*u.PasswordHash, oldPwd) {
		writeError(c, http.StatusUnauthorized, "unauthorized", "invalid password")
		return
	}

	hash, err := appauth.HashPassword(newPwd)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "hash password failed", err)
		return
	}
	u, err = h.st.ChangePassword(ctx, uid, hash)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":          u.ID,
			"openid":      valueOrEmpty(u.WeChatOpenID),
			"nickname":    u.WeChatNickname,
			"avatarUrl":   u.WeChatAvatarURL,
			"username":    valueOrEmpty(u.Username),
			"hasPassword": u.PasswordHash != nil && strings.TrimSpace(*u.PasswordHash) != "",
			"wechatBound": u.WeChatOpenID != nil && strings.TrimSpace(*u.WeChatOpenID) != "",
		},
	})
}

type bindWeChatRequest struct {
	Code string `json:"code"`
}

func (h *AuthHandlers) BindWeChat(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	var req bindWeChatRequest
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
	u, err := h.st.BindWeChatOpenID(ctx, uid, openid)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		if err == store.ErrConflict {
			writeError(c, http.StatusConflict, "conflict", "wechat already bound")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":          u.ID,
			"openid":      valueOrEmpty(u.WeChatOpenID),
			"nickname":    u.WeChatNickname,
			"avatarUrl":   u.WeChatAvatarURL,
			"username":    valueOrEmpty(u.Username),
			"hasPassword": u.PasswordHash != nil && strings.TrimSpace(*u.PasswordHash) != "",
			"wechatBound": u.WeChatOpenID != nil && strings.TrimSpace(*u.WeChatOpenID) != "",
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

func valueOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func normalizeUsername(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func validateUsername(username string) error {
	if username == "" {
		return &appError{message: "username required"}
	}
	if len(username) < 3 || len(username) > 32 {
		return &appError{message: "username length must be 3-32"}
	}
	for _, ch := range username {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' {
			continue
		}
		return &appError{message: "username can only contain a-z, 0-9 and _"}
	}
	return nil
}

func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return &appError{message: "password required"}
	}
	if len(password) < 8 || len(password) > 72 {
		return &appError{message: "password length must be 8-72"}
	}
	return nil
}

func isSchemaOutdated(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42703" {
		// undefined_column
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, `column "username" does not exist`) ||
		strings.Contains(msg, `column "password_hash" does not exist`) ||
		strings.Contains(msg, `password_set_at`)
}
