package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	appconfig "scorehub/internal/config"
)

type LocationHandlers struct {
	cfg appconfig.Config
}

func NewLocationHandlers(cfg appconfig.Config) *LocationHandlers {
	return &LocationHandlers{cfg: cfg}
}

func (h *LocationHandlers) ReverseGeocode(ctx context.Context, c *app.RequestContext) {
	latStr := strings.TrimSpace(string(c.Query("lat")))
	lngStr := strings.TrimSpace(string(c.Query("lng")))
	if latStr == "" || lngStr == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "lat/lng required")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid lat")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid lng")
		return
	}

	fallback := fmt.Sprintf("%.4f,%.4f", lat, lng)
	hasAnyKey := h.cfg.TencentMapKey != "" || h.cfg.AmapKey != "" || h.cfg.BaiduMapAK != ""
	if !hasAnyKey {
		c.JSON(http.StatusOK, map[string]any{"locationText": fallback, "source": "raw"})
		return
	}

	type chosenProvider struct {
		source string
		call   func(context.Context) (string, error)
	}

	// 根据各家 QPS 限制选择一个可用的 provider；每次请求只调用一家，避免“依次调用”浪费配额。
	var chosen *chosenProvider
	if h.cfg.TencentMapKey != "" && tencentGeocodeQPSLimiter.Allow() {
		chosen = &chosenProvider{
			source: "tencent",
			call: func(ctx context.Context) (string, error) {
				return reverseGeocodeTencent(ctx, h.cfg.TencentMapKey, lat, lng)
			},
		}
	} else if h.cfg.AmapKey != "" && amapGeocodeQPSLimiter.Allow() {
		chosen = &chosenProvider{
			source: "amap",
			call: func(ctx context.Context) (string, error) {
				return reverseGeocodeAmap(ctx, h.cfg.AmapKey, lat, lng)
			},
		}
	} else if h.cfg.BaiduMapAK != "" && baiduGeocodeQPSLimiter.Allow() {
		chosen = &chosenProvider{
			source: "baidu",
			call: func(ctx context.Context) (string, error) {
				return reverseGeocodeBaidu(ctx, h.cfg.BaiduMapAK, lat, lng)
			},
		}
	}

	if chosen == nil {
		c.JSON(http.StatusOK, map[string]any{
			"locationText": fallback,
			"source":       "raw",
			"geocodeError": "geocode rate limited",
		})
		return
	}

	text, err := chosen.call(ctx)
	if err != nil || strings.TrimSpace(text) == "" {
		out := map[string]any{"locationText": fallback, "source": "raw"}
		if err != nil {
			out["geocodeError"] = err.Error()
		}
		c.JSON(http.StatusOK, out)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"locationText": text, "source": chosen.source})
}

type tencentGeocoderResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
		Address          string `json:"address"`
		AddressComponent struct {
			Nation   string `json:"nation"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"address_component"`
		FormattedAddresses struct {
			Recommend string `json:"recommend"`
			Rough     string `json:"rough"`
		} `json:"formatted_addresses"`
	} `json:"result"`
}

func reverseGeocodeTencent(ctx context.Context, key string, lat, lng float64) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "apis.map.qq.com",
		Path:   "/ws/geocoder/v1/",
	}
	q := u.Query()
	q.Set("location", fmt.Sprintf("%.6f,%.6f", lat, lng))
	q.Set("key", key)
	q.Set("get_poi", "0")
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
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tencent map http %d", resp.StatusCode)
	}

	var r tencentGeocoderResp
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if r.Status != 0 {
		msg := strings.TrimSpace(r.Message)
		if msg == "" {
			msg = "unknown error"
		}
		return "", fmt.Errorf("tencent map status %d: %s", r.Status, msg)
	}

	if v := strings.TrimSpace(r.Result.FormattedAddresses.Recommend); v != "" {
		return v, nil
	}

	if v := strings.TrimSpace(r.Result.FormattedAddresses.Rough); v != "" {
		return v, nil
	}

	if addr := strings.TrimSpace(r.Result.Address); addr != "" {
		return addr, nil
	}

	ac := r.Result.AddressComponent
	city := strings.TrimSpace(ac.City)
	province := strings.TrimSpace(ac.Province)
	district := strings.TrimSpace(ac.District)

	main := city
	if main == "" {
		main = province
	}
	parts := make([]string, 0, 2)
	if main != "" {
		parts = append(parts, main)
	}
	if district != "" && district != main {
		parts = append(parts, district)
	}
	out := strings.Join(parts, "·")
	if out != "" {
		return out, nil
	}

	return "", nil
}

type amapRegeoResp struct {
	Status    string `json:"status"`
	Info      string `json:"info"`
	Infocode  string `json:"infocode"`
	Regeocode struct {
		FormattedAddress string `json:"formatted_address"`
		AddressComponent struct {
			Province string          `json:"province"`
			City     json.RawMessage `json:"city"`
			District string          `json:"district"`
		} `json:"addressComponent"`
	} `json:"regeocode"`
}

func reverseGeocodeAmap(ctx context.Context, key string, lat, lng float64) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "restapi.amap.com",
		Path:   "/v3/geocode/regeo",
	}
	q := u.Query()
	q.Set("location", fmt.Sprintf("%.6f,%.6f", lng, lat)) // 高德：lng,lat
	q.Set("key", key)
	q.Set("output", "JSON")
	q.Set("extensions", "base")
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
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("amap http %d", resp.StatusCode)
	}

	var r amapRegeoResp
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if strings.TrimSpace(r.Status) != "1" {
		msg := strings.TrimSpace(r.Info)
		if msg == "" {
			msg = "unknown error"
		}
		code := strings.TrimSpace(r.Infocode)
		if code != "" {
			return "", fmt.Errorf("amap status %s (%s): %s", r.Status, code, msg)
		}
		return "", fmt.Errorf("amap status %s: %s", r.Status, msg)
	}

	if v := strings.TrimSpace(r.Regeocode.FormattedAddress); v != "" {
		return v, nil
	}

	ac := r.Regeocode.AddressComponent
	city := strings.TrimSpace(parseAmapCity(ac.City))
	province := strings.TrimSpace(ac.Province)
	district := strings.TrimSpace(ac.District)

	main := city
	if main == "" {
		main = province
	}
	parts := make([]string, 0, 2)
	if main != "" {
		parts = append(parts, main)
	}
	if district != "" && district != main {
		parts = append(parts, district)
	}
	short := strings.Join(parts, "·")
	if short != "" {
		return short, nil
	}

	return "", nil
}

func parseAmapCity(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return strings.TrimSpace(strings.Join(arr, ""))
	}
	return ""
}

type baiduReverseGeocodeResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Msg     string `json:"msg"`
	Result  struct {
		FormattedAddress string `json:"formatted_address"`
		AddressComponent struct {
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
		} `json:"addressComponent"`
	} `json:"result"`
}

func reverseGeocodeBaidu(ctx context.Context, ak string, lat, lng float64) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.map.baidu.com",
		Path:   "/reverse_geocoding/v3/",
	}
	q := u.Query()
	q.Set("location", fmt.Sprintf("%.6f,%.6f", lat, lng)) // 百度：lat,lng
	q.Set("coordtype", "gcj02ll")                         // uni.getLocation(type=gcj02)
	q.Set("output", "json")
	q.Set("extensions_poi", "0")
	q.Set("ak", ak)
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
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("baidu map http %d", resp.StatusCode)
	}

	var r baiduReverseGeocodeResp
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if r.Status != 0 {
		msg := strings.TrimSpace(r.Message)
		if msg == "" {
			msg = strings.TrimSpace(r.Msg)
		}
		if msg == "" {
			msg = "unknown error"
		}
		return "", fmt.Errorf("baidu map status %d: %s", r.Status, msg)
	}

	if v := strings.TrimSpace(r.Result.FormattedAddress); v != "" {
		return v, nil
	}

	ac := r.Result.AddressComponent
	city := strings.TrimSpace(ac.City)
	province := strings.TrimSpace(ac.Province)
	district := strings.TrimSpace(ac.District)

	main := city
	if main == "" {
		main = province
	}
	parts := make([]string, 0, 2)
	if main != "" {
		parts = append(parts, main)
	}
	if district != "" && district != main {
		parts = append(parts, district)
	}
	out := strings.Join(parts, "·")
	if out != "" {
		return out, nil
	}

	return "", nil
}

type tokenBucket struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(rate, burst float64) *tokenBucket {
	if rate < 0 {
		rate = 0
	}
	if burst < 1 {
		burst = 1
	}
	now := time.Now()
	return &tokenBucket{
		rate:  rate,
		burst: burst,
		tokens: func() float64 {
			if rate > 0 {
				return math.Min(burst, rate)
			}
			return burst
		}(),
		last: now,
	}
}

func (b *tokenBucket) Allow() bool {
	if b == nil {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if b.last.IsZero() {
		b.last = now
	}

	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 && b.rate > 0 {
		b.tokens = math.Min(b.burst, b.tokens+elapsed*b.rate)
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}

var (
	// QPS 约束：参考 README 中的「并发 x 次/秒」说明。
	tencentGeocodeQPSLimiter = newTokenBucket(5, 5)
	amapGeocodeQPSLimiter    = newTokenBucket(3, 3)
	baiduGeocodeQPSLimiter   = newTokenBucket(3, 3)
)
