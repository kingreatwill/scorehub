package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"scorehub/internal/http/middleware"
	"scorehub/internal/store"
)

type MeHandlers struct {
	st *store.Store
}

func NewMeHandlers(st *store.Store) *MeHandlers {
	return &MeHandlers{st: st}
}

func (h *MeHandlers) GetMe(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
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

	c.JSON(http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":        u.ID,
			"openid":    u.WeChatOpenID,
			"nickname":  u.WeChatNickname,
			"avatarUrl": u.WeChatAvatarURL,
		},
	})
}

type updateMeRequest struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatarUrl"`
}

func (h *MeHandlers) UpdateMe(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req updateMeRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if req.Nickname == nil && req.AvatarURL == nil {
		writeError(c, http.StatusBadRequest, "bad_request", "nickname or avatarUrl required")
		return
	}

	if req.Nickname != nil {
		v := strings.TrimSpace(*req.Nickname)
		req.Nickname = &v
	}
	if req.AvatarURL != nil {
		v := strings.TrimSpace(*req.AvatarURL)
		req.AvatarURL = &v
	}

	u, err := h.st.UpdateUserProfile(ctx, uid, req.Nickname, req.AvatarURL)
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
			"id":        u.ID,
			"openid":    u.WeChatOpenID,
			"nickname":  u.WeChatNickname,
			"avatarUrl": u.WeChatAvatarURL,
		},
	})
}
