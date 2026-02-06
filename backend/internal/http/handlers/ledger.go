package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	appauth "scorehub/internal/auth"
	appconfig "scorehub/internal/config"
	"scorehub/internal/http/middleware"
	"scorehub/internal/store"
)

type LedgerHandlers struct {
	cfg appconfig.Config
	st  *store.Store
}

func NewLedgerHandlers(cfg appconfig.Config, st *store.Store) *LedgerHandlers {
	return &LedgerHandlers{cfg: cfg, st: st}
}

type createLedgerRequest struct {
	Name string `json:"name"`
}

type updateLedgerRequest struct {
	Name          *string `json:"name"`
	ShareDisabled *bool   `json:"shareDisabled"`
}

type bindLedgerMemberRequest struct {
	MemberID  string `json:"memberId"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

func (h *LedgerHandlers) CreateLedger(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req createLedgerRequest
	if body, err := c.Body(); err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
			return
		}
	} else if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = time.Now().Format("2006-01-02 15:04") + " 记账"
	}

	user, err := h.st.GetUserByID(ctx, uid)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	ledger, member, err := h.st.CreateLedger(ctx, user, name)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"ledger": toLedgerDTO(ledger),
		"member": toLedgerMemberDTO(member),
	})
}

func (h *LedgerHandlers) ListLedgers(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	items, err := h.st.ListLedgersForUser(ctx, uid)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, it := range items {
		out = append(out, map[string]any{
			"id":          it.LedgerID,
			"name":        it.Name,
			"createdAt":   it.StartTime,
			"updatedAt":   it.UpdatedAt,
			"status":      it.Status,
			"endedAt":     it.EndedAt,
			"memberCount": it.MemberCount,
			"recordCount": it.RecordCount,
		})
	}

	c.JSON(http.StatusOK, map[string]any{"items": out})
}

func (h *LedgerHandlers) GetLedgerDetail(ctx context.Context, c *app.RequestContext) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	ledger, members, records, err := h.st.GetLedgerDetail(ctx, id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "ledger not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	uid, ok := optionalUserID(c, h.cfg)
	isOwner := ok && ledger.CreatedByUserID == uid

	remarkByMember := map[string]string{}
	if isOwner {
		for _, m := range members {
			remark := strings.TrimSpace(m.Remark)
			if remark != "" {
				remarkByMember[m.ID] = remark
			}
		}
	}

	var memOut []any
	for _, m := range members {
		if !isOwner {
			m.Remark = ""
		}
		memOut = append(memOut, toLedgerMemberDTO(m))
	}
	var recOut []any
	for _, r := range records {
		if !isOwner {
			if r.Type == "remark" {
				continue
			}
			r.Note = ""
			recOut = append(recOut, toLedgerRecordDTO(r))
			continue
		}
		if r.Note == "" {
			if remark := remarkByMember[r.MemberID]; remark != "" {
				r.Note = remark
			}
		}
		recOut = append(recOut, toLedgerRecordDTO(r))
	}

	c.JSON(http.StatusOK, map[string]any{
		"ledger":  toLedgerDTO(ledger),
		"members": memOut,
		"records": recOut,
	})
}

func optionalUserID(c *app.RequestContext, cfg appconfig.Config) (int64, bool) {
	if uid, ok := middleware.UserID(c); ok {
		return uid, true
	}
	authHeader := strings.TrimSpace(string(c.GetHeader("Authorization")))
	if authHeader == "" {
		return 0, false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return 0, false
	}
	token := strings.TrimSpace(authHeader[len(prefix):])
	if token == "" {
		return 0, false
	}
	uid, err := appauth.ParseToken([]byte(cfg.TokenSecret), token)
	if err != nil {
		return 0, false
	}
	return uid, true
}

func (h *LedgerHandlers) UpdateLedger(ctx context.Context, c *app.RequestContext) {
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

	var req updateLedgerRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if req.Name == nil && req.ShareDisabled == nil {
		writeError(c, http.StatusBadRequest, "bad_request", "name or shareDisabled required")
		return
	}
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "name required")
			return
		}
		req.Name = &trimmed
	}

	ledger, err := h.st.UpdateLedger(ctx, id, uid, req.Name, req.ShareDisabled)
	if err != nil {
		switch err {
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "forbidden", "no permission")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "ledger not found")
			return
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"ledger": toLedgerDTO(ledger)})
}

func (h *LedgerHandlers) BindLedgerMember(ctx context.Context, c *app.RequestContext) {
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

	var req bindLedgerMemberRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	memberID := strings.TrimSpace(req.MemberID)
	if memberID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "memberId required")
		return
	}

	member, err := h.st.BindLedgerMember(ctx, id, uid, memberID, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
	if err != nil {
		switch err {
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "ledger member not found")
			return
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "share_disabled", "share disabled")
			return
		case store.ErrConflict:
			writeError(c, http.StatusConflict, "conflict", "member already bound")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "ledger ended")
			return
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid member")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"member": toLedgerMemberDTO(member)})
}

func (h *LedgerHandlers) GetInviteQRCode(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	ledgerID := strings.TrimSpace(c.Param("id"))
	if ledgerID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	ledger, err := h.st.GetLedger(ctx, ledgerID)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "ledger not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	if ledger.CreatedByUserID != uid {
		writeError(c, http.StatusForbidden, "forbidden", "no permission")
		return
	}
	if ledger.ShareDisabled {
		writeError(c, http.StatusForbidden, "share_disabled", "share disabled")
		return
	}
	if ledger.Status != "recording" {
		writeError(c, http.StatusBadRequest, "ended", "ledger ended")
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
	img, err := getWeChatMiniProgramCode(ctx, accessToken, ledger.InviteCode, "pages/join/index")
	if err != nil {
		var we *wechatAPIError
		if errors.As(err, &we) && we.Code == 41030 {
			img, err = getWeChatMiniProgramCode(ctx, accessToken, ledger.InviteCode, "")
		}
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "wechat qrcode failed", err)
		return
	}

	c.Data(http.StatusOK, "image/png", img)
}

func (h *LedgerHandlers) AddLedgerMember(ctx context.Context, c *app.RequestContext) {
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

	var req addLedgerMemberRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	m, err := h.st.AddLedgerMember(ctx, id, uid, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL), strings.TrimSpace(req.Remark))
	if err != nil {
		switch err {
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "forbidden", "no permission")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "ledger not found")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "ledger ended")
			return
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid nickname")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"member": toLedgerMemberDTO(m)})
}

type addLedgerRecordRequest struct {
	MemberID string  `json:"memberId"`
	Type     string  `json:"type"`
	Amount   float64 `json:"amount"`
	Note     string  `json:"note"`
}

type updateLedgerMemberRequest struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Remark    string `json:"remark"`
}

type addLedgerMemberRequest struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Remark    string `json:"remark"`
}

func (h *LedgerHandlers) AddLedgerRecord(ctx context.Context, c *app.RequestContext) {
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

	var req addLedgerRecordRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if strings.TrimSpace(req.MemberID) == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "memberId required")
		return
	}

	amount := req.Amount
	if !validTwoDecimals(amount) {
		writeError(c, http.StatusBadRequest, "bad_request", "amount must have at most 2 decimals")
		return
	}
	t := strings.ToLower(strings.TrimSpace(req.Type))
	if t != "expense" && t != "income" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid type")
		return
	}

	r, err := h.st.AddLedgerRecord(ctx, id, uid, strings.TrimSpace(req.MemberID), t, amount, strings.TrimSpace(req.Note))
	if err != nil {
		switch err {
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "forbidden", "no permission")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "ledger not found")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "ledger ended")
			return
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid record")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"record": toLedgerRecordDTO(r)})
}

func (h *LedgerHandlers) UpdateLedgerMember(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	ledgerID := strings.TrimSpace(c.Param("id"))
	memberID := strings.TrimSpace(c.Param("memberId"))
	if ledgerID == "" || memberID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "id required")
		return
	}

	var req updateLedgerMemberRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	m, err := h.st.UpdateLedgerMember(ctx, ledgerID, uid, memberID, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL), strings.TrimSpace(req.Remark))
	if err != nil {
		switch err {
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "forbidden", "no permission")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "ledger member not found")
			return
		case store.ErrScorebookEnded:
			writeError(c, http.StatusBadRequest, "ended", "ledger ended")
			return
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid nickname")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"member": toLedgerMemberDTO(m)})
}

func (h *LedgerHandlers) EndLedger(ctx context.Context, c *app.RequestContext) {
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

	ledger, err := h.st.EndLedger(ctx, id, uid)
	if err != nil {
		switch err {
		case store.ErrForbidden:
			writeError(c, http.StatusForbidden, "forbidden", "no permission")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "ledger not found")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"ledger": toLedgerDTO(ledger)})
}

func toLedgerDTO(sb store.Scorebook) map[string]any {
	return map[string]any{
		"id":              sb.ID,
		"name":            sb.Name,
		"createdAt":       sb.StartTime,
		"updatedAt":       sb.UpdatedAt,
		"status":          sb.Status,
		"endedAt":         sb.EndedAt,
		"inviteCode":      sb.InviteCode,
		"createdByUserId": sb.CreatedByUserID,
		"shareDisabled":   sb.ShareDisabled,
	}
}

func toLedgerMemberDTO(m store.LedgerMember) map[string]any {
	var userID any
	if m.UserID != nil {
		userID = *m.UserID
	}
	return map[string]any{
		"id":        m.ID,
		"userId":    userID,
		"role":      m.Role,
		"nickname":  m.Nickname,
		"avatarUrl": m.AvatarURL,
		"remark":    m.Remark,
		"score":     m.Score,
		"createdAt": m.CreatedAt,
		"updatedAt": m.UpdatedAt,
	}
}

func toLedgerRecordDTO(r store.LedgerRecord) map[string]any {
	return map[string]any{
		"id":           r.ID,
		"memberId":     r.MemberID,
		"fromMemberId": r.FromMemberID,
		"toMemberId":   r.ToMemberID,
		"type":         r.Type,
		"amount":       r.Amount,
		"note":         r.Note,
		"createdAt":    r.CreatedAt,
	}
}
