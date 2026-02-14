package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"scorehub/internal/http/middleware"
	"scorehub/internal/store"
)

type BirthdayHandlers struct {
	st *store.Store
}

func NewBirthdayHandlers(st *store.Store) *BirthdayHandlers {
	return &BirthdayHandlers{st: st}
}

type createBirthdayRequest struct {
	Name          string `json:"name"`
	Gender        string `json:"gender"`
	Phone         string `json:"phone"`
	Relation      string `json:"relation"`
	Note          string `json:"note"`
	AvatarURL     string `json:"avatarUrl"`
	SolarBirthday string `json:"solarBirthday"`
	LunarBirthday string `json:"lunarBirthday"`
	PrimaryType   string `json:"primaryType"`
	PrimaryMonth  int    `json:"primaryMonth"`
	PrimaryDay    int    `json:"primaryDay"`
	PrimaryYear   int    `json:"primaryYear"`
}

type updateBirthdayRequest struct {
	Name          *string `json:"name"`
	Gender        *string `json:"gender"`
	Phone         *string `json:"phone"`
	Relation      *string `json:"relation"`
	Note          *string `json:"note"`
	AvatarURL     *string `json:"avatarUrl"`
	SolarBirthday *string `json:"solarBirthday"`
	LunarBirthday *string `json:"lunarBirthday"`
	PrimaryType   *string `json:"primaryType"`
	PrimaryMonth  *int    `json:"primaryMonth"`
	PrimaryDay    *int    `json:"primaryDay"`
	PrimaryYear   *int    `json:"primaryYear"`
}

func (h *BirthdayHandlers) CreateBirthday(ctx context.Context, c *app.RequestContext) {
	uid, ok := middleware.UserID(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req createBirthdayRequest
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
		writeError(c, http.StatusBadRequest, "bad_request", "name required")
		return
	}
	gender := strings.TrimSpace(req.Gender)
	if gender != "" && gender != "男" && gender != "女" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid gender")
		return
	}
	primaryType := normalizePrimaryType(req.PrimaryType)

	solar, err := parseDatePtr(req.SolarBirthday)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid solarBirthday")
		return
	}
	lunar := strings.TrimSpace(req.LunarBirthday)

	primaryMonth := req.PrimaryMonth
	primaryDay := req.PrimaryDay
	primaryYear := req.PrimaryYear
	if primaryType == "solar" {
		if solar == nil {
			writeError(c, http.StatusBadRequest, "bad_request", "solarBirthday required")
			return
		}
		primaryMonth = int(solar.Month())
		primaryDay = solar.Day()
		primaryYear = 0
	} else {
		if lunar == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "lunarBirthday required")
			return
		}
		if primaryMonth == 0 || primaryDay == 0 {
			writeError(c, http.StatusBadRequest, "bad_request", "primaryMonth/primaryDay required")
			return
		}
		if primaryYear == 0 {
			primaryYear = time.Now().Year()
		}
	}
	if err := validateMonthDay(primaryMonth, primaryDay); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	contact, err := h.st.CreateBirthdayContact(ctx, uid, store.BirthdayContactInput{
		Name:          name,
		Gender:        gender,
		Phone:         strings.TrimSpace(req.Phone),
		Relation:      strings.TrimSpace(req.Relation),
		Note:          strings.TrimSpace(req.Note),
		AvatarURL:     strings.TrimSpace(req.AvatarURL),
		SolarBirthday: solar,
		LunarBirthday: lunar,
		PrimaryType:   primaryType,
		PrimaryMonth:  primaryMonth,
		PrimaryDay:    primaryDay,
		PrimaryYear:   primaryYear,
	})
	if err != nil {
		if err == store.ErrInvalidArgument {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid name")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"birthday": toBirthdayDTO(contact)})
}

func (h *BirthdayHandlers) ListBirthdays(ctx context.Context, c *app.RequestContext) {
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

	items, err := h.st.ListBirthdayContacts(ctx, uid, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	var out []any
	for _, it := range items {
		out = append(out, toBirthdayListDTO(it))
	}
	c.JSON(http.StatusOK, map[string]any{"items": out, "limit": limit, "offset": offset})
}

func (h *BirthdayHandlers) GetBirthday(ctx context.Context, c *app.RequestContext) {
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

	contact, err := h.st.GetBirthdayContact(ctx, uid, id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "birthday not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"birthday": toBirthdayDTO(contact)})
}

func (h *BirthdayHandlers) UpdateBirthday(ctx context.Context, c *app.RequestContext) {
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

	var req updateBirthdayRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			writeError(c, http.StatusBadRequest, "bad_request", "name required")
			return
		}
		req.Name = &name
	}
	if req.Gender != nil {
		g := strings.TrimSpace(*req.Gender)
		if g != "" && g != "男" && g != "女" {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid gender")
			return
		}
		req.Gender = &g
	}
	if req.Phone != nil {
		phone := strings.TrimSpace(*req.Phone)
		req.Phone = &phone
	}
	if req.Relation != nil {
		rel := strings.TrimSpace(*req.Relation)
		req.Relation = &rel
	}
	if req.Note != nil {
		note := strings.TrimSpace(*req.Note)
		req.Note = &note
	}
	if req.AvatarURL != nil {
		avatar := strings.TrimSpace(*req.AvatarURL)
		req.AvatarURL = &avatar
	}

	var solar *time.Time
	solarSetNull := false
	if req.SolarBirthday != nil {
		val := strings.TrimSpace(*req.SolarBirthday)
		if val == "" {
			solarSetNull = true
		} else {
			parsed, err := parseDatePtr(val)
			if err != nil || parsed == nil {
				writeError(c, http.StatusBadRequest, "bad_request", "invalid solarBirthday")
				return
			}
			solar = parsed
		}
	}

	var primaryType *string
	if req.PrimaryType != nil {
		pt := normalizePrimaryType(*req.PrimaryType)
		primaryType = &pt
	}

	if req.PrimaryMonth != nil && req.PrimaryDay != nil {
		if err := validateMonthDay(*req.PrimaryMonth, *req.PrimaryDay); err != nil {
			writeError(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	}
	if (req.PrimaryMonth != nil && req.PrimaryDay == nil) || (req.PrimaryMonth == nil && req.PrimaryDay != nil) {
		writeError(c, http.StatusBadRequest, "bad_request", "primaryMonth/primaryDay required")
		return
	}

	if primaryType != nil && *primaryType == "solar" && solar != nil && (req.PrimaryMonth == nil || req.PrimaryDay == nil) {
		m := int(solar.Month())
		d := solar.Day()
		req.PrimaryMonth = &m
		req.PrimaryDay = &d
	}
	if (primaryType != nil && *primaryType == "lunar") && (req.PrimaryMonth == nil || req.PrimaryDay == nil) {
		writeError(c, http.StatusBadRequest, "bad_request", "primaryMonth/primaryDay required")
		return
	}
	if req.LunarBirthday != nil && (req.PrimaryMonth == nil || req.PrimaryDay == nil) {
		writeError(c, http.StatusBadRequest, "bad_request", "primaryMonth/primaryDay required")
		return
	}
	if req.LunarBirthday != nil && strings.TrimSpace(*req.LunarBirthday) == "" {
		empty := ""
		req.LunarBirthday = &empty
	}

	update := store.BirthdayContactUpdate{
		Name:          req.Name,
		Gender:        req.Gender,
		Phone:         req.Phone,
		Relation:      req.Relation,
		Note:          req.Note,
		AvatarURL:     req.AvatarURL,
		SolarBirthday: solar,
		SolarSetNull:  solarSetNull,
		LunarBirthday: req.LunarBirthday,
		PrimaryType:   primaryType,
		PrimaryMonth:  req.PrimaryMonth,
		PrimaryDay:    req.PrimaryDay,
		PrimaryYear:   req.PrimaryYear,
	}

	contact, err := h.st.UpdateBirthdayContact(ctx, uid, id, update)
	if err != nil {
		switch err {
		case store.ErrInvalidArgument:
			writeError(c, http.StatusBadRequest, "bad_request", "invalid payload")
			return
		case store.ErrNotFound:
			writeError(c, http.StatusNotFound, "not_found", "birthday not found")
			return
		default:
			writeError(c, http.StatusInternalServerError, "internal", "db error", err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]any{"birthday": toBirthdayDTO(contact)})
}

func (h *BirthdayHandlers) DeleteBirthday(ctx context.Context, c *app.RequestContext) {
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

	if err := h.st.DeleteBirthdayContact(ctx, uid, id); err != nil {
		if err == store.ErrNotFound {
			writeError(c, http.StatusNotFound, "not_found", "birthday not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal", "db error", err)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func toBirthdayDTO(c store.BirthdayContact) map[string]any {
	return map[string]any{
		"id":            c.ID,
		"userId":        c.UserID,
		"name":          c.Name,
		"gender":        c.Gender,
		"phone":         c.Phone,
		"relation":      c.Relation,
		"note":          c.Note,
		"avatarUrl":     c.AvatarURL,
		"solarBirthday": formatDate(c.SolarBirthday),
		"lunarBirthday": c.LunarBirthday,
		"primaryType":   c.PrimaryType,
		"primaryMonth":  c.PrimaryMonth,
		"primaryDay":    c.PrimaryDay,
		"primaryYear":   c.PrimaryYear,
		"createdAt":     c.CreatedAt,
		"updatedAt":     c.UpdatedAt,
	}
}

func toBirthdayListDTO(c store.BirthdayContactWithDays) map[string]any {
	out := toBirthdayDTO(c.BirthdayContact)
	out["daysLeft"] = c.DaysLeft
	out["nextBirthday"] = formatDate(&c.NextBirthday)
	return out
}

func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func parseDatePtr(raw string) (*time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func normalizePrimaryType(v string) string {
	t := strings.ToLower(strings.TrimSpace(v))
	if t == "lunar" {
		return "lunar"
	}
	return "solar"
}

func validateMonthDay(month, day int) error {
	if month < 1 || month > 12 {
		return errors.New("invalid primaryMonth")
	}
	if day < 1 || day > 31 {
		return errors.New("invalid primaryDay")
	}
	return nil
}
