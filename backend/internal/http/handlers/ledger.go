package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"scorehub/internal/http/middleware"
	"scorehub/internal/store"
)

type LedgerHandlers struct {
	st *store.Store
}

func NewLedgerHandlers(st *store.Store) *LedgerHandlers {
	return &LedgerHandlers{st: st}
}

type createLedgerRequest struct {
	Name string `json:"name"`
}

type updateLedgerRequest struct {
	Name string `json:"name"`
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

	var memOut []any
	for _, m := range members {
		memOut = append(memOut, toLedgerMemberDTO(m))
	}
	var recOut []any
	for _, r := range records {
		recOut = append(recOut, toLedgerRecordDTO(r))
	}

	c.JSON(http.StatusOK, map[string]any{
		"ledger":  toLedgerDTO(ledger),
		"members": memOut,
		"records": recOut,
	})
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
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "name required")
		return
	}

	ledger, err := h.st.UpdateLedgerName(ctx, id, uid, name)
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

	m, err := h.st.AddLedgerMember(ctx, id, uid, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
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

	m, err := h.st.UpdateLedgerMember(ctx, ledgerID, uid, memberID, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.AvatarURL))
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
		"id":        sb.ID,
		"name":      sb.Name,
		"createdAt": sb.StartTime,
		"updatedAt": sb.UpdatedAt,
		"status":    sb.Status,
		"endedAt":   sb.EndedAt,
	}
}

func toLedgerMemberDTO(m store.LedgerMember) map[string]any {
	return map[string]any{
		"id":        m.ID,
		"role":      m.Role,
		"nickname":  m.Nickname,
		"avatarUrl": m.AvatarURL,
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
