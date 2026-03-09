package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"scorehub/internal/http/middleware"
	"scorehub/internal/tts"
)

type VoiceHandlers struct {
	tts *tts.Service
}

func NewVoiceHandlers(ttsService *tts.Service) *VoiceHandlers {
	return &VoiceHandlers{tts: ttsService}
}

type scoreReceivedSpeechRequest struct {
	Delta    float64 `json:"delta"`
	Text     string  `json:"text"`
	Voice    string  `json:"voice"`
	VoiceKey string  `json:"voiceKey"`
}

func (h *VoiceHandlers) ScoreReceivedSpeech(ctx context.Context, c *app.RequestContext) {
	if _, ok := middleware.UserID(c); !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	var req scoreReceivedSpeechRequest
	body, err := c.Body()
	if err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "read body failed")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	voice := normalizeVoiceID(req.Voice)
	if voice == "" {
		voice = normalizeVoiceID(req.VoiceKey)
	}

	textInputProvided := strings.TrimSpace(req.Text) != ""
	text := normalizeVoiceText(req.Text)
	if text == "" && textInputProvided {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid text")
		return
	}
	if text == "" {
		if req.Delta <= 0 || !validVoiceDelta(req.Delta) {
			writeError(c, http.StatusBadRequest, "bad_request", "invalid delta")
			return
		}
		text = tts.BuildReceivedScoreText(req.Delta)
	}
	if text == "" {
		writeError(c, http.StatusBadRequest, "bad_request", "invalid delta")
		return
	}

	audio, contentType, err := h.tts.GenerateSpeech(ctx, text, voice)
	if err != nil {
		switch {
		case errors.Is(err, tts.ErrNotConfigured):
			writeError(c, http.StatusServiceUnavailable, "tts_not_configured", "tts not configured")
		case errors.Is(err, tts.ErrInvalidVoice):
			writeError(c, http.StatusBadRequest, "bad_request", "invalid voice")
		default:
			writeError(c, http.StatusBadGateway, "tts_failed", "tts generation failed", err)
		}
		return
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Data(http.StatusOK, contentType, audio)
}

func (h *VoiceHandlers) ListVoices(ctx context.Context, c *app.RequestContext) {
	if _, ok := middleware.UserID(c); !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}

	voices, err := h.tts.ListVoices(ctx)
	if err != nil {
		switch {
		case errors.Is(err, tts.ErrNotConfigured):
			writeError(c, http.StatusServiceUnavailable, "tts_not_configured", "tts not configured")
		default:
			writeError(c, http.StatusBadGateway, "tts_failed", "tts voices failed", err)
		}
		return
	}

	c.JSON(http.StatusOK, map[string]any{
		"items": voices,
	})
}

func validVoiceDelta(v float64) bool {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return math.Abs(v*100-math.Round(v*100)) < 1e-6
}

func normalizeVoiceText(raw string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if text == "" {
		return ""
	}
	if len([]rune(text)) > 80 {
		return ""
	}
	return text
}

func normalizeVoiceID(raw string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if text == "" {
		return ""
	}
	if len([]rune(text)) > 120 {
		return ""
	}
	return text
}
