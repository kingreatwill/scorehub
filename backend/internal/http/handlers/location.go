package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	if h.cfg.TencentMapKey == "" {
		c.JSON(http.StatusOK, map[string]any{"locationText": fallback, "source": "raw"})
		return
	}

	text, err := reverseGeocodeTencent(ctx, h.cfg.TencentMapKey, lat, lng)
	if err != nil || strings.TrimSpace(text) == "" {
		out := map[string]any{"locationText": fallback, "source": "raw"}
		if err != nil {
			out["geocodeError"] = err.Error()
		}
		c.JSON(http.StatusOK, out)
		return
	}

	c.JSON(http.StatusOK, map[string]any{"locationText": text, "source": "tencent"})
}

type tencentGeocoderResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Result  struct {
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

	if v := strings.TrimSpace(r.Result.FormattedAddresses.Rough); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(r.Result.FormattedAddresses.Recommend); v != "" {
		return v, nil
	}
	return "", nil
}
