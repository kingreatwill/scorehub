package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"

	appauth "scorehub/internal/auth"
	appconfig "scorehub/internal/config"
	"scorehub/internal/http/middleware"
	"scorehub/internal/realtime"
	"scorehub/internal/store"
)

type ScorebookHandlers struct {
	cfg appconfig.Config
	st  *store.Store
	hub *realtime.Hub

	upgrader websocket.HertzUpgrader
}

func NewScorebookHandlers(cfg appconfig.Config, st *store.Store, hub *realtime.Hub) *ScorebookHandlers {
	return &ScorebookHandlers{
		cfg: cfg,
		st:  st,
		hub: hub,
		upgrader: websocket.HertzUpgrader{
			CheckOrigin: func(ctx *app.RequestContext) bool { return true },
		},
	}
}

type createScorebookRequest struct {
	Name         string `json:"name"`
	LocationText string `json:"locationText"`
	BookType     string `json:"bookType"`
}

func (h *ScorebookHandlers) CreateScorebook(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req createScorebookRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	name := strings.TrimSpace(req.Name)
	locationText := strings.TrimSpace(req.LocationText)
	bookType := "scorebook"
	if name == "" {
		ts := time.Now().Format("2006-01-02 15:04")
		if locationText != "" {
			name = ts + " " + locationText
		} else {
			name = ts
		}
	}

	user, err := h.st.GetUserByID(ctx, uid)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	sb, owner, err := h.st.CreateScorebook(ctx, user, name, locationText, bookType)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	h.hub.Broadcast(sb.ID, map[string]any{
		"type": "scorebook.created",
		"data": map[string]any{"id": sb.ID},
	})

	c.JSON(http.StatusOK, map[string]any{
		"scorebook": toScorebookDTO(sb),
		"me":        toMemberDTO(owner, 0, owner.ID),
	})
}

func (h *ScorebookHandlers) ListMyScorebooks(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	items, err := h.st.ListScorebooksForUser(ctx, uid)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, it := range items {
		out = append(out, map[string]any{
			"id":           it.ScorebookID,
			"name":         it.Name,
			"locationText": it.LocationText,
			"startTime":    it.StartTime,
			"updatedAt":    it.UpdatedAt,
			"status":       it.Status,
			"bookType":     it.BookType,
			"endedAt":      it.EndedAt,
			"inviteCode":   it.InviteCode,
			"isOwner":      it.MyRole == "owner",
			"memberCount":  it.MemberCount,
		})
	}

	c.JSON(http.StatusOK, map[string]any{"items": out})
}

func (h *ScorebookHandlers) GetScorebookDetail(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	sb, myMemberID, myRole, members, err := h.st.GetScorebookDetail(ctx, id, uid)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "scorebook not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var memOut []any
	for _, m := range members {
		memOut = append(memOut, toMemberDTO(m.Member, m.Score, myMemberID))
	}

	c.JSON(http.StatusOK, map[string]any{
		"scorebook": toScorebookDTO(sb),
		"me": map[string]any{
			"memberId": myMemberID,
			"isOwner":  myRole == "owner",
		},
		"members": memOut,
	})
}

type updateScorebookRequest struct {
	Name string `json:"name"`
}

func (h *ScorebookHandlers) UpdateScorebook(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	var req updateScorebookRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "name required")
		return
	}

	sb, err := h.st.UpdateScorebookName(ctx, id, uid, name)
	if err != nil {
		if err == store.ErrForbidden {
			writeError(c, http.StatusForbidden, "forbidden", "only owner can update")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	h.hub.Broadcast(sb.ID, map[string]any{
		"type": "scorebook.updated",
		"data": map[string]any{"id": sb.ID, "name": sb.Name, "updatedAt": sb.UpdatedAt},
	})

	c.JSON(http.StatusOK, map[string]any{"scorebook": toScorebookDTO(sb)})
}

func (h *ScorebookHandlers) EndScorebook(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	sb, err := h.st.EndScorebook(ctx, id, uid)
	if err != nil {
		if err == store.ErrForbidden {
			writeError(c, http.StatusForbidden, "forbidden", "only owner can end")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var champion any
	var runnerUp any
	var third any
	if tops, err := h.st.GetTopWinners(ctx, sb.ID); err == nil {
		if len(tops) >= 1 {
			champion = map[string]any{
				"memberId":  tops[0].ID,
				"nickname":  tops[0].Nickname,
				"avatarUrl": tops[0].AvatarURL,
				"score":     tops[0].Score,
			}
		}
		if len(tops) >= 2 {
			runnerUp = map[string]any{
				"memberId":  tops[1].ID,
				"nickname":  tops[1].Nickname,
				"avatarUrl": tops[1].AvatarURL,
				"score":     tops[1].Score,
			}
		}
		if len(tops) >= 3 {
			third = map[string]any{
				"memberId":  tops[2].ID,
				"nickname":  tops[2].Nickname,
				"avatarUrl": tops[2].AvatarURL,
				"score":     tops[2].Score,
			}
		}
	} else {
		// 不影响结束流程，但记录内部错误方便排查
		c.Error(err)
	}
	winners := map[string]any{
		"champion": champion,
		"runnerUp": runnerUp,
		"third":    third,
	}

	h.hub.Broadcast(sb.ID, map[string]any{
		"type": "scorebook.ended",
		"data": map[string]any{"id": sb.ID, "endedAt": sb.EndedAt, "updatedAt": sb.UpdatedAt, "winners": winners},
	})

	c.JSON(http.StatusOK, map[string]any{"scorebook": toScorebookDTO(sb), "winners": winners})
}

type joinScorebookRequest struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func (h *ScorebookHandlers) JoinScorebook(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	scorebookID := strings.TrimSpace(c.Param("id"))
	if scorebookID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	var req joinScorebookRequest
	if body, err := c.Body(); err == nil && len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	user, err := h.st.GetUserByID(ctx, uid)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	m, err := h.st.JoinScorebook(ctx, scorebookID, user, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
	if err != nil {
		switch err {
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "scorebook not found")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "scorebook ended")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	h.hub.Broadcast(scorebookID, map[string]any{
		"type": "member.joined",
		"data": map[string]any{
			"member": map[string]any{
				"id":        m.ID,
				"nickname":  m.Nickname,
				"avatarUrl": m.AvatarURL,
				"role":      m.Role,
				"joinedAt":  m.JoinedAt,
			},
		},
	})

	c.JSON(http.StatusOK, map[string]any{"member": toMemberDTO(m, 0, m.ID)})
}

func (h *ScorebookHandlers) UpdateMyProfile(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	scorebookID := strings.TrimSpace(c.Param("id"))
	if scorebookID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	var req joinScorebookRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	nickname := strings.TrimSpace(req.Nickname)
	if nickname == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "nickname required")
		return
	}

	m, err := h.st.UpdateMyProfile(ctx, scorebookID, uid, nickname, strings.TrimSpace(req.AvatarURL))
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "member not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	h.hub.Broadcast(scorebookID, map[string]any{
		"type": "member.updated",
		"data": map[string]any{
			"member": map[string]any{
				"id":        m.ID,
				"nickname":  m.Nickname,
				"avatarUrl": m.AvatarURL,
				"updatedAt": m.UpdatedAt,
			},
		},
	})

	c.JSON(http.StatusOK, map[string]any{"member": toMemberDTO(m, 0, m.ID)})
}

type createRecordRequest struct {
	ToMemberID string  `json:"toMemberId"`
	Delta      float64 `json:"delta"`
	Note       string  `json:"note"`
}

func (h *ScorebookHandlers) CreateRecord(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	scorebookID := strings.TrimSpace(c.Param("id"))
	if scorebookID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	var req createRecordRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	req.ToMemberID = strings.TrimSpace(req.ToMemberID)
	req.Note = strings.TrimSpace(req.Note)
	if req.ToMemberID == "" || req.Delta <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "toMemberId and positive delta required")
		return
	}
	if !validTwoDecimals(req.Delta) {
		writeError(c, http.StatusBadRequest, "bad_request", "delta must have at most 2 decimals")
		return
	}

	r, err := h.st.CreateRecord(ctx, scorebookID, uid, req.ToMemberID, req.Delta, req.Note)
	if err != nil {
		switch err {
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "not found")
			return
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "forbidden", "not a member")
			return
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid member")
			return
		case store.ErrInvalidDelta:
			writeError(c, http.StatusBadRequest, "bad_request", "delta must be positive")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "scorebook ended")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	h.hub.Broadcast(scorebookID, map[string]any{
		"type": "record.created",
		"data": map[string]any{
			"record": map[string]any{
				"id":           r.ID,
				"fromMemberId": r.FromMemberID,
				"toMemberId":   r.ToMemberID,
				"delta":        r.Delta,
				"note":         r.Note,
				"createdAt":    r.CreatedAt,
			},
		},
	})

	c.JSON(http.StatusOK, map[string]any{"record": map[string]any{
		"id":           r.ID,
		"fromMemberId": r.FromMemberID,
		"toMemberId":   r.ToMemberID,
		"delta":        r.Delta,
		"note":         r.Note,
		"createdAt":    r.CreatedAt,
	}})
}

func (h *ScorebookHandlers) ListRecords(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	scorebookID := strings.TrimSpace(c.Param("id"))
	if scorebookID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	limit := int32(50)
	offset := int32(0)
	if v := strings.TrimSpace(string(c.Query("limit"))); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = int32(n)
		}
	}
	if v := strings.TrimSpace(string(c.Query("offset"))); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}

	items, err := h.st.ListRecords(ctx, scorebookID, uid, limit, offset)
	if err != nil {
		if err == store.ErrForbidden {
			writeError(c, http.StatusForbidden, "forbidden", "not a member")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, r := range items {
		out = append(out, map[string]any{
			"id":           r.ID,
			"fromMemberId": r.FromMemberID,
			"toMemberId":   r.ToMemberID,
			"delta":        r.Delta,
			"note":         r.Note,
			"createdAt":    r.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, map[string]any{"items": out, "limit": limit, "offset": offset})
}

func (h *ScorebookHandlers) GetInviteQRCode(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	scorebookID := strings.TrimSpace(c.Param("id"))
	if scorebookID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	isMember, err := h.st.IsMember(ctx, scorebookID, uid)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	if !isMember {
		writeError(c, http.StatusForbidden, "forbidden", "not a member")
		return
	}

	sb, err := h.st.GetScorebook(ctx, scorebookID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "scorebook not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	if sb.Status != "recording" {
		writeError(c, http.StatusBadRequest, "ended", "scorebook ended")
		return
	}

	if strings.TrimSpace(h.cfg.WeChatAppID) == "" || strings.TrimSpace(h.cfg.WeChatSecret) == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "wechat appid/secret not configured")
		return
	}

	accessToken, err := getWeChatAccessToken(ctx, h.cfg.WeChatAppID, h.cfg.WeChatSecret)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "wechat token failed", err)
		return
	}
	img, err := getWeChatMiniProgramCode(ctx, accessToken, sb.InviteCode, "pages/join/index")
	if err != nil {
		var we *wechatAPIError
		if errors.As(err, &we) && we.Code == 41030 {
			// join 页面尚未发布到线上版本时，会报 invalid page（41030）；回退为首页 + scene
			img, err = getWeChatMiniProgramCode(ctx, accessToken, sb.InviteCode, "")
		}
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "wechat qrcode failed", err)
		return
	}

	c.Data(http.StatusOK, "image/png", img)
}

func (h *ScorebookHandlers) GetInviteInfo(ctx context.Context, c *app.RequestContext) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "code required")
		return
	}

	info, err := h.st.GetInviteInfo(ctx, code)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "invite not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	bookType := strings.ToLower(strings.TrimSpace(info.BookType))
	if bookType == "" {
		bookType = "scorebook"
	}
	bookID := strings.TrimSpace(info.BookID)
	if bookType == "ledger" && info.ShareDisabled {
		writeError(c, http.StatusForbidden, "share_disabled", "share disabled")
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"invite": map[string]any{
			"code":      code,
			"bookType":  bookType,
			"bookId":    bookID,
			"name":      info.Name,
			"status":    info.Status,
			"shareDisabled": info.ShareDisabled,
			"updatedAt": info.UpdatedAt,
			// backwards compatibility for existing clients
			"scorebookId": func() string {
				if bookType == "scorebook" {
					return bookID
				}
				return ""
			}(),
			"ledgerId": func() string {
				if bookType == "ledger" {
					return bookID
				}
				return ""
			}(),
		},
	})
}

func (h *ScorebookHandlers) JoinByInviteCode(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "code required")
		return
	}

	scorebookID, err := h.st.ScorebookIDByInviteCode(ctx, code)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "invite not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var req joinScorebookRequest
	if body, err := c.Body(); err == nil && len(body) > 0 {
		_ = json.Unmarshal(body, &req)
	}

	user, err := h.st.GetUserByID(ctx, uid)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	m, err := h.st.JoinScorebook(ctx, scorebookID, user, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
	if err != nil {
		switch err {
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "scorebook not found")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "scorebook ended")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	h.hub.Broadcast(scorebookID, map[string]any{
		"type": "member.joined",
		"data": map[string]any{
			"member": map[string]any{
				"id":        m.ID,
				"nickname":  m.Nickname,
				"avatarUrl": m.AvatarURL,
				"role":      m.Role,
				"joinedAt":  m.JoinedAt,
			},
		},
	})

	c.JSON(http.StatusOK, map[string]any{
		"scorebookId": scorebookID,
		"member":      toMemberDTO(m, 0, m.ID),
	})
}

func (h *ScorebookHandlers) ScorebookWS(ctx context.Context, c *app.RequestContext) {
	scorebookID := strings.TrimSpace(c.Param("id"))
	if scorebookID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	uid, ok := middleware.UserID(c)
	if !ok {
		token := extractTokenForWS(c)
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		id, err := appauth.ParseToken([]byte(h.cfg.TokenSecret), token)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		uid = id
	}

	isMember, err := h.st.IsMember(ctx, scorebookID, uid)
	if err != nil || !isMember {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	h.upgrader.Upgrade(c, func(conn *websocket.Conn) {
		h.hub.Join(scorebookID, conn)
		defer h.hub.Leave(scorebookID, conn)
		defer conn.Close()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
}

func extractTokenForWS(c *app.RequestContext) string {
	if v := strings.TrimSpace(string(c.Query("token"))); v != "" {
		return v
	}
	return ""
}

func toScorebookDTO(sb store.Scorebook) map[string]any {
	return map[string]any{
		"id":           sb.ID,
		"name":         sb.Name,
		"locationText": sb.LocationText,
		"startTime":    sb.StartTime,
		"updatedAt":    sb.UpdatedAt,
		"status":       sb.Status,
		"bookType":     sb.BookType,
		"endedAt":      sb.EndedAt,
		"inviteCode":   sb.InviteCode,
	}
}

func normalizeBookType(v string) string {
	t := strings.ToLower(strings.TrimSpace(v))
	switch t {
	case "ledger", "scorebook":
		return t
	default:
		return "scorebook"
	}
}

func validTwoDecimals(v float64) bool {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return math.Abs(v*100-math.Round(v*100)) < 1e-6
}

func toMemberDTO(m store.Member, score float64, myMemberID string) map[string]any {
	return map[string]any{
		"id":        m.ID,
		"nickname":  m.Nickname,
		"avatarUrl": m.AvatarURL,
		"role":      m.Role,
		"joinedAt":  m.JoinedAt,
		"updatedAt": m.UpdatedAt,
		"score":     score,
		"isMe":      m.ID == myMemberID,
		"isOwner":   m.Role == "owner",
	}
}
