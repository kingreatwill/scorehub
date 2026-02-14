package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"scorehub/internal/http/middleware"
	"scorehub/internal/store"
)

type DepositHandlers struct {
	st *store.Store
}

func NewDepositHandlers(st *store.Store) *DepositHandlers {
	return &DepositHandlers{st: st}
}

type createDepositAccountRequest struct {
	Bank      string `json:"bank"`
	Branch    string `json:"branch"`
	AccountNo string `json:"accountNo"`
	Holder    string `json:"holder"`
	AvatarURL string `json:"avatarUrl"`
	Note      string `json:"note"`
}

type updateDepositAccountRequest struct {
	Bank      *string `json:"bank"`
	Branch    *string `json:"branch"`
	AccountNo *string `json:"accountNo"`
	Holder    *string `json:"holder"`
	AvatarURL *string `json:"avatarUrl"`
	Note      *string `json:"note"`
}

type createDepositRecordRequest struct {
	Currency    string                    `json:"currency"`
	Amount      float64                   `json:"amount"`
	AmountUpper string                    `json:"amountUpper"`
	TermValue   int                       `json:"termValue"`
	TermUnit    string                    `json:"termUnit"`
	Rate        float64                   `json:"rate"`
	StartDate   string                    `json:"startDate"`
	EndDate     string                    `json:"endDate"`
	Interest    float64                   `json:"interest"`
	ReceiptNo   string                    `json:"receiptNo"`
	Status      string                    `json:"status"`
	WithdrawnAt string                    `json:"withdrawnAt"`
	Tags        []string                  `json:"tags"`
	Attachments []store.DepositAttachment `json:"attachments"`
	Note        string                    `json:"note"`
}

type updateDepositRecordRequest struct {
	Currency    *string                    `json:"currency"`
	Amount      *float64                   `json:"amount"`
	AmountUpper *string                    `json:"amountUpper"`
	TermValue   *int                       `json:"termValue"`
	TermUnit    *string                    `json:"termUnit"`
	Rate        *float64                   `json:"rate"`
	StartDate   *string                    `json:"startDate"`
	EndDate     *string                    `json:"endDate"`
	Interest    *float64                   `json:"interest"`
	ReceiptNo   *string                    `json:"receiptNo"`
	Status      *string                    `json:"status"`
	WithdrawnAt *string                    `json:"withdrawnAt"`
	Tags        *[]string                  `json:"tags"`
	Attachments *[]store.DepositAttachment `json:"attachments"`
	Note        *string                    `json:"note"`
}

func (h *DepositHandlers) CreateDepositAccount(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req createDepositAccountRequest
	if body, err := c.Body(); err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
			return
		}
	} else if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}

	bank := strings.TrimSpace(req.Bank)
	if bank == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "bank required")
		return
	}

	account, err := h.st.CreateDepositAccount(ctx, uid, store.DepositAccountInput{
		Bank:      bank,
		Branch:    strings.TrimSpace(req.Branch),
		AccountNo: strings.TrimSpace(req.AccountNo),
		Holder:    strings.TrimSpace(req.Holder),
		AvatarURL: strings.TrimSpace(req.AvatarURL),
		Note:      strings.TrimSpace(req.Note),
	})
	if err != nil {
		if err == store.ErrInvalidArgument {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid bank")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"account": toDepositAccountDTO(account)})
}

func (h *DepositHandlers) ListDepositAccounts(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	limit := int32(20)
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

	items, err := h.st.ListDepositAccounts(ctx, uid, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, it := range items {
		out = append(out, toDepositAccountDTO(it))
	}
	c.JSON(http.StatusOK, map[string]any{"items": out, "limit": limit, "offset": offset})
}

func (h *DepositHandlers) GetDepositAccount(ctx context.Context, c *app.RequestContext) {
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

	account, err := h.st.GetDepositAccount(ctx, uid, id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "account not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"account": toDepositAccountDTO(account)})
}

func (h *DepositHandlers) UpdateDepositAccount(ctx context.Context, c *app.RequestContext) {
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

	var req updateDepositAccountRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	if req.Bank != nil {
		val := strings.TrimSpace(*req.Bank)
		if val == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "bank required")
			return
		}
		req.Bank = &val
	}
	trimPtr(&req.Branch)
	trimPtr(&req.AccountNo)
	trimPtr(&req.Holder)
	trimPtr(&req.AvatarURL)
	trimPtr(&req.Note)

	account, err := h.st.UpdateDepositAccount(ctx, uid, id, store.DepositAccountUpdate{
		Bank:      req.Bank,
		Branch:    req.Branch,
		AccountNo: req.AccountNo,
		Holder:    req.Holder,
		AvatarURL: req.AvatarURL,
		Note:      req.Note,
	})
	if err != nil {
		switch err {
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "account not found")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"account": toDepositAccountDTO(account)})
}

func (h *DepositHandlers) DeleteDepositAccount(ctx context.Context, c *app.RequestContext) {
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

	if err := h.st.DeleteDepositAccount(ctx, uid, id); err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "account not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *DepositHandlers) CreateDepositRecord(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "accountId required")
		return
	}

	var req createDepositRecordRequest
	if body, err := c.Body(); err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
			return
		}
	} else if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}

	currency := normalizeCurrency(req.Currency)
	if !isValidCurrency(currency) {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid currency")
		return
	}
	if req.Amount <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "amount required")
		return
	}
	if req.TermValue <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "termValue required")
		return
	}
	termUnit := normalizeTermUnit(req.TermUnit)
	if termUnit == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid termUnit")
		return
	}
	if req.Rate <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "rate required")
		return
	}
	startDate, err := parseDateRequired(req.StartDate)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid startDate")
		return
	}
	endDate, err := parseDateRequired(req.EndDate)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid endDate")
		return
	}

	status := normalizeStatus(req.Status)
	if status == "" {
		status = "未到期"
	}
	if status != "未到期" && status != "已到期" && status != "已支取" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
		return
	}

	var withdrawnAt *time.Time
	if strings.TrimSpace(req.WithdrawnAt) != "" {
		t, err := parseDateRequired(req.WithdrawnAt)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid withdrawnAt")
			return
		}
		withdrawnAt = &t
	}
	if status == "已支取" && withdrawnAt == nil {
		t := endDate
		withdrawnAt = &t
	}
	if status != "已支取" {
		withdrawnAt = nil
	}

	record, err := h.st.CreateDepositRecord(ctx, uid, store.DepositRecordInput{
		AccountID:   accountID,
		Currency:    currency,
		Amount:      req.Amount,
		AmountUpper: strings.TrimSpace(req.AmountUpper),
		TermValue:   req.TermValue,
		TermUnit:    termUnit,
		Rate:        req.Rate,
		StartDate:   startDate,
		EndDate:     endDate,
		Interest:    req.Interest,
		ReceiptNo:   strings.TrimSpace(req.ReceiptNo),
		Status:      status,
		WithdrawnAt: withdrawnAt,
		Tags:        normalizeTags(req.Tags),
		Attachments: normalizeAttachments(req.Attachments),
		Note:        strings.TrimSpace(req.Note),
	})
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "account not found")
			return
		}
		if err == store.ErrInvalidArgument {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"record": toDepositRecordDTO(record)})
}

func (h *DepositHandlers) ListDepositRecords(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	accountID := strings.TrimSpace(c.Query("accountId"))
	status := normalizeStatus(c.Query("status"))
	tags := parseTagsQuery(c.Query("tags"))
	if status == "全部" {
		status = ""
	}
	if status != "" && status != "未到期" && status != "已到期" && status != "已支取" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
		return
	}

	limit := int32(20)
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

	items, err := h.st.ListDepositRecords(ctx, uid, accountID, status, tags, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, it := range items {
		out = append(out, toDepositRecordDTO(it))
	}
	c.JSON(http.StatusOK, map[string]any{"items": out, "limit": limit, "offset": offset})
}

func (h *DepositHandlers) ListDepositAccountRecords(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	accountID := strings.TrimSpace(c.Param("id"))
	if accountID == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "accountId required")
		return
	}

	status := normalizeStatus(c.Query("status"))
	tags := parseTagsQuery(c.Query("tags"))
	if status == "全部" {
		status = ""
	}
	if status != "" && status != "未到期" && status != "已到期" && status != "已支取" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
		return
	}

	limit := int32(20)
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

	items, err := h.st.ListDepositRecords(ctx, uid, accountID, status, tags, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, it := range items {
		out = append(out, toDepositRecordDTO(it))
	}
	c.JSON(http.StatusOK, map[string]any{"items": out, "limit": limit, "offset": offset})
}

func (h *DepositHandlers) GetDepositRecord(ctx context.Context, c *app.RequestContext) {
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

	record, err := h.st.GetDepositRecord(ctx, uid, id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "record not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	c.JSON(http.StatusOK, map[string]any{"record": toDepositRecordDTO(record)})
}

func (h *DepositHandlers) UpdateDepositRecord(ctx context.Context, c *app.RequestContext) {
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

	var req updateDepositRecordRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	if req.Currency != nil {
		cur := normalizeCurrency(*req.Currency)
		if !isValidCurrency(cur) {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid currency")
			return
		}
		req.Currency = &cur
	}
	if req.Amount != nil && *req.Amount <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid amount")
		return
	}
	if req.TermValue != nil && *req.TermValue <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid termValue")
		return
	}
	if req.TermUnit != nil {
		unit := normalizeTermUnit(*req.TermUnit)
		if unit == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid termUnit")
			return
		}
		req.TermUnit = &unit
	}
	if req.Rate != nil && *req.Rate <= 0 {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid rate")
		return
	}

	var startDate *time.Time
	if req.StartDate != nil {
		val := strings.TrimSpace(*req.StartDate)
		if val == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid startDate")
			return
		}
		t, err := parseDateRequired(val)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid startDate")
			return
		}
		startDate = &t
	}
	var endDate *time.Time
	if req.EndDate != nil {
		val := strings.TrimSpace(*req.EndDate)
		if val == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid endDate")
			return
		}
		t, err := parseDateRequired(val)
		if err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid endDate")
			return
		}
		endDate = &t
	}
	if req.AmountUpper != nil {
		val := strings.TrimSpace(*req.AmountUpper)
		req.AmountUpper = &val
	}
	if req.ReceiptNo != nil {
		val := strings.TrimSpace(*req.ReceiptNo)
		req.ReceiptNo = &val
	}
	if req.Note != nil {
		val := strings.TrimSpace(*req.Note)
		req.Note = &val
	}

	var status *string
	var withdrawnAt *time.Time
	withdrawnSetNull := false
	if req.Status != nil {
		s := normalizeStatus(*req.Status)
		if s == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
			return
		}
		if s != "未到期" && s != "已到期" && s != "已支取" {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
			return
		}
		status = &s
	}
	if req.WithdrawnAt != nil {
		val := strings.TrimSpace(*req.WithdrawnAt)
		if val == "" {
			withdrawnSetNull = true
		} else {
			t, err := parseDateRequired(val)
			if err != nil {
				writeError(c, http.StatusBadRequest, "bad_request", "invalid withdrawnAt")
				return
			}
			withdrawnAt = &t
		}
	}
	if status != nil && *status == "已支取" && withdrawnAt == nil && !withdrawnSetNull {
		now := time.Now()
		t := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		withdrawnAt = &t
		if endDate == nil {
			endDate = &t
		}
	}
	if status != nil && *status != "已支取" && req.WithdrawnAt != nil {
		withdrawnSetNull = true
		withdrawnAt = nil
	}

	var tags *[]string
	if req.Tags != nil {
		normalized := normalizeTags(*req.Tags)
		tags = &normalized
	}
	var attachments *[]store.DepositAttachment
	if req.Attachments != nil {
		normalized := normalizeAttachments(*req.Attachments)
		attachments = &normalized
	}

	record, err := h.st.UpdateDepositRecord(ctx, uid, id, store.DepositRecordUpdate{
		Currency:         req.Currency,
		Amount:           req.Amount,
		AmountUpper:      req.AmountUpper,
		TermValue:        req.TermValue,
		TermUnit:         req.TermUnit,
		Rate:             req.Rate,
		StartDate:        startDate,
		EndDate:          endDate,
		Interest:         req.Interest,
		ReceiptNo:        req.ReceiptNo,
		Status:           status,
		WithdrawnAt:      withdrawnAt,
		WithdrawnSetNull: withdrawnSetNull,
		Tags:             tags,
		Attachments:      attachments,
		Note:             req.Note,
	})
	if err != nil {
		switch err {
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "record not found")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"record": toDepositRecordDTO(record)})
}

func (h *DepositHandlers) DeleteDepositRecord(ctx context.Context, c *app.RequestContext) {
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
	if err := h.st.DeleteDepositRecord(ctx, uid, id); err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "record not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (h *DepositHandlers) ListDepositTags(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	accountID := strings.TrimSpace(c.Query("accountId"))
	status := normalizeStatus(c.Query("status"))
	if status == "全部" {
		status = ""
	}
	if status != "" && status != "未到期" && status != "已到期" && status != "已支取" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
		return
	}

	items, err := h.st.ListDepositTags(ctx, uid, accountID, status)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}
	var out []any
	for _, it := range items {
		out = append(out, map[string]any{
			"tag":   it.Tag,
			"count": it.Count,
		})
	}
	c.JSON(http.StatusOK, map[string]any{"items": out})
}

func (h *DepositHandlers) GetDepositStats(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	accountID := strings.TrimSpace(c.Query("accountId"))
	status := normalizeStatus(c.Query("status"))
	tags := parseTagsQuery(c.Query("tags"))
	if status == "全部" {
		status = ""
	}
	if status != "" && status != "未到期" && status != "已到期" && status != "已支取" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid status")
		return
	}

	stats, err := h.st.GetDepositStats(ctx, uid, accountID, status, tags)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var totals []any
	for _, item := range stats.Totals {
		totals = append(totals, map[string]any{
			"currency": item.Currency,
			"amount":   item.Amount,
		})
	}
	var annualYields []any
	for _, item := range stats.AnnualYields {
		annualYields = append(annualYields, map[string]any{
			"currency": item.Currency,
			"amount":   item.Amount,
		})
	}
	var accountTotals []any
	for _, item := range stats.AccountTotals {
		accountTotals = append(accountTotals, map[string]any{
			"accountId": item.AccountID,
			"currency":  item.Currency,
			"amount":    item.Amount,
		})
	}

	c.JSON(http.StatusOK, map[string]any{
		"stats": map[string]any{
			"totals":        totals,
			"annualYields":  annualYields,
			"accountTotals": accountTotals,
		},
	})
}

func toDepositAccountDTO(a store.DepositAccount) map[string]any {
	return map[string]any{
		"id":        a.ID,
		"userId":    a.UserID,
		"bank":      a.Bank,
		"branch":    a.Branch,
		"accountNo": a.AccountNo,
		"holder":    a.Holder,
		"avatarUrl": a.AvatarURL,
		"note":      a.Note,
		"createdAt": a.CreatedAt,
		"updatedAt": a.UpdatedAt,
	}
}

func toDepositRecordDTO(r store.DepositRecord) map[string]any {
	return map[string]any{
		"id":          r.ID,
		"userId":      r.UserID,
		"accountId":   r.AccountID,
		"currency":    r.Currency,
		"amount":      r.Amount,
		"amountUpper": r.AmountUpper,
		"termValue":   r.TermValue,
		"termUnit":    r.TermUnit,
		"rate":        r.Rate,
		"startDate":   formatDateString(r.StartDate),
		"endDate":     formatDateString(r.EndDate),
		"interest":    r.Interest,
		"receiptNo":   r.ReceiptNo,
		"status":      r.Status,
		"withdrawnAt": formatDatePtr(r.WithdrawnAt),
		"tags":        r.Tags,
		"attachments": r.Attachments,
		"note":        r.Note,
		"createdAt":   r.CreatedAt,
		"updatedAt":   r.UpdatedAt,
	}
}

func normalizeCurrency(raw string) string {
	v := strings.TrimSpace(strings.ToUpper(raw))
	if v == "" {
		return "CNY"
	}
	return v
}

func isValidCurrency(v string) bool {
	return v == "CNY" || v == "USD"
}

func normalizeTermUnit(raw string) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	if v == "year" || v == "month" {
		return v
	}
	return ""
}

func normalizeStatus(raw string) string {
	return strings.TrimSpace(raw)
}

func parseDateRequired(raw string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(raw))
}

func formatDateString(t time.Time) string {
	return t.Format("2006-01-02")
}

func formatDatePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func trimPtr(v **string) {
	if v == nil || *v == nil {
		return
	}
	val := strings.TrimSpace(**v)
	*v = &val
}

func normalizeTags(raw []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, tag := range raw {
		v := strings.TrimSpace(tag)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func parseTagsQuery(raw string) []string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	return normalizeTags(parts)
}

func normalizeAttachments(raw []store.DepositAttachment) []store.DepositAttachment {
	var out []store.DepositAttachment
	for _, item := range raw {
		t := strings.TrimSpace(item.Type)
		if t != "image" && t != "file" {
			continue
		}
		u := strings.TrimSpace(item.URL)
		if u == "" {
			continue
		}
		out = append(out, store.DepositAttachment{
			Type: t,
			URL:  u,
			Name: strings.TrimSpace(item.Name),
		})
	}
	return out
}
