package tts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	appconfig "scorehub/internal/config"
)

var (
	ErrNotConfigured = errors.New("tts not configured")
	ErrInvalidVoice  = errors.New("invalid voice")
)

const (
	audioCacheTTL = 24 * time.Hour
	voiceCacheTTL = 10 * time.Minute
)

type VoiceOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Language    string `json:"language,omitempty"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"isDefault,omitempty"`
}

type voiceBinding struct {
	VoiceOption
	ProviderVoice string
}

type Service struct {
	client *http.Client
	cfg    config

	staticVoices []voiceBinding

	mu         sync.RWMutex
	audioCache map[string]cachedAudio
	voiceCache cachedVoices
}

type config struct {
	apiBase      string
	speechPath   string
	voicesPath   string
	voicesJSON   string
	apiKey       string
	model        string
	format       string
	defaultVoice string
}

type cachedAudio struct {
	body        []byte
	contentType string
	expiresAt   time.Time
}

type cachedVoices struct {
	items     []voiceBinding
	expiresAt time.Time
}

type providerVoiceEnvelope struct {
	Voices       []providerVoiceItem `json:"voices"`
	Data         []providerVoiceItem `json:"data"`
	Results      []providerVoiceItem `json:"results"`
	DefaultVoice string              `json:"defaultVoice"`
}

type providerVoiceItem struct {
	ID          string `json:"id"`
	Voice       string `json:"voice"`
	URI         string `json:"uri"`
	Name        string `json:"name"`
	CustomName  string `json:"customName"`
	Label       string `json:"label"`
	DisplayName string `json:"displayName"`
	Language    string `json:"language"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
	IsDefault   bool   `json:"isDefault"`
}

type configuredVoiceItem struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Language      string `json:"language"`
	Description   string `json:"description"`
	IsDefault     bool   `json:"isDefault"`
	ProviderVoice string `json:"providerVoice"`
	Voice         string `json:"voice"`
	Name          string `json:"name"`
}

func New(cfg appconfig.Config) *Service {
	service := &Service{
		client: &http.Client{Timeout: 20 * time.Second},
		cfg: config{
			apiBase:      strings.TrimSpace(cfg.TTSAPIBase),
			speechPath:   strings.TrimSpace(cfg.TTSAPISpeechPath),
			voicesPath:   strings.TrimSpace(cfg.TTSAPIVoicesPath),
			voicesJSON:   strings.TrimSpace(cfg.TTSVoicesJSON),
			apiKey:       strings.TrimSpace(cfg.TTSAPIKey),
			model:        strings.TrimSpace(cfg.TTSModel),
			format:       normalizeFormat(cfg.TTSAudioFormat),
			defaultVoice: strings.TrimSpace(cfg.TTSDefaultVoice),
		},
		audioCache: make(map[string]cachedAudio),
	}
	service.staticVoices = parseConfiguredVoices(service.cfg.voicesJSON, service.cfg.defaultVoice)
	return service
}

func normalizeFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "wav":
		return "wav"
	default:
		return "mp3"
	}
}

func (s *Service) Configured() bool {
	if s == nil {
		return false
	}
	return s.cfg.apiBase != "" && s.cfg.model != ""
}

func (s *Service) GenerateReceivedScore(ctx context.Context, delta float64, voice string) ([]byte, string, error) {
	return s.GenerateSpeech(ctx, BuildReceivedScoreText(delta), voice)
}

func (s *Service) GenerateSpeech(ctx context.Context, text string, voice string) ([]byte, string, error) {
	if !s.Configured() {
		return nil, "", ErrNotConfigured
	}

	resolvedVoice, err := s.resolveVoice(ctx, voice)
	if err != nil {
		return nil, "", err
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil, "", fmt.Errorf("empty speech text")
	}
	cacheKey := cacheKeyFor(resolvedVoice, text, s.cfg.format)
	if body, contentType, ok := s.getCachedAudio(cacheKey); ok {
		return body, contentType, nil
	}

	body, contentType, err := s.requestSpeech(ctx, text, resolvedVoice)
	if err != nil {
		return nil, "", err
	}
	s.putCachedAudio(cacheKey, body, contentType)
	return body, contentType, nil
}

func (s *Service) ListVoices(ctx context.Context) ([]VoiceOption, error) {
	if !s.Configured() {
		return nil, ErrNotConfigured
	}
	if len(s.staticVoices) > 0 {
		return bindingOptions(s.staticVoices), nil
	}

	if items, ok := s.getCachedVoices(); ok {
		return bindingOptions(items), nil
	}

	items, err := s.requestVoices(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrInvalidVoice
	}
	s.putCachedVoices(items)
	return bindingOptions(items), nil
}

func (s *Service) resolveVoice(ctx context.Context, raw string) (string, error) {
	requested := strings.TrimSpace(raw)
	if len(s.staticVoices) > 0 {
		return resolveVoiceBinding(requested, s.staticVoices)
	}

	if items, ok := s.getCachedVoices(); ok {
		return resolveVoiceBinding(requested, items)
	}

	items, err := s.requestVoices(ctx)
	if err != nil {
		if requested != "" {
			return requested, nil
		}
		return "", err
	}
	if len(items) == 0 {
		return "", ErrInvalidVoice
	}
	s.putCachedVoices(items)
	return resolveVoiceBinding(requested, items)
}

func BuildReceivedScoreText(delta float64) string {
	if delta <= 0 {
		return "收到分数"
	}
	value := fmt.Sprintf("%.2f", delta)
	value = strings.TrimSuffix(value, ".00")
	value = strings.TrimSuffix(value, "0")
	value = strings.TrimSuffix(value, ".")
	return "收到" + value + "分"
}

func cacheKeyFor(voice, text, format string) string {
	sum := sha256.Sum256([]byte(voice + "\n" + text + "\n" + format))
	return hex.EncodeToString(sum[:])
}

func (s *Service) getCachedAudio(key string) ([]byte, string, bool) {
	now := time.Now()
	s.mu.RLock()
	item, ok := s.audioCache[key]
	s.mu.RUnlock()
	if !ok || now.After(item.expiresAt) {
		if ok {
			s.mu.Lock()
			delete(s.audioCache, key)
			s.mu.Unlock()
		}
		return nil, "", false
	}
	return bytes.Clone(item.body), item.contentType, true
}

func (s *Service) putCachedAudio(key string, body []byte, contentType string) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, item := range s.audioCache {
		if now.After(item.expiresAt) {
			delete(s.audioCache, k)
		}
	}
	s.audioCache[key] = cachedAudio{
		body:        bytes.Clone(body),
		contentType: contentType,
		expiresAt:   now.Add(audioCacheTTL),
	}
}

func (s *Service) getCachedVoices() ([]voiceBinding, bool) {
	now := time.Now()
	s.mu.RLock()
	cache := s.voiceCache
	s.mu.RUnlock()
	if len(cache.items) == 0 || now.After(cache.expiresAt) {
		return nil, false
	}
	return slices.Clone(cache.items), true
}

func (s *Service) putCachedVoices(items []voiceBinding) {
	s.mu.Lock()
	s.voiceCache = cachedVoices{
		items:     slices.Clone(items),
		expiresAt: time.Now().Add(voiceCacheTTL),
	}
	s.mu.Unlock()
}

func (s *Service) requestVoices(ctx context.Context) ([]voiceBinding, error) {
	endpoint, err := joinEndpoint(s.cfg.apiBase, s.cfg.voicesPath, "/audio/voices")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if s.cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 240 {
			msg = msg[:240]
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("tts voices provider http %d: %s", resp.StatusCode, msg)
	}

	var envelope providerVoiceEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	rawItems := envelope.Voices
	if len(rawItems) == 0 {
		rawItems = envelope.Data
	}
	if len(rawItems) == 0 {
		rawItems = envelope.Results
	}

	out := make([]voiceBinding, 0, len(rawItems))
	for _, item := range rawItems {
		id := firstNonEmpty(item.ID, item.Voice, item.URI, item.Name, item.CustomName, item.Label, item.DisplayName)
		if id == "" {
			continue
		}
		label := firstNonEmpty(item.Label, item.DisplayName, item.Name, item.CustomName, item.Voice, item.ID)
		providerVoice := firstNonEmpty(item.Voice, item.URI, item.ID)
		out = append(out, voiceBinding{
			VoiceOption: VoiceOption{
				ID:          strings.TrimSpace(id),
				Label:       strings.TrimSpace(label),
				Language:    strings.TrimSpace(item.Language),
				Description: strings.TrimSpace(item.Description),
				IsDefault:   item.Default || item.IsDefault,
			},
			ProviderVoice: strings.TrimSpace(providerVoice),
		})
	}
	return normalizeVoiceBindings(out, envelope.DefaultVoice, s.cfg.defaultVoice), nil
}

func parseConfiguredVoices(raw string, fallbackDefault string) []voiceBinding {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var payload []configuredVoiceItem
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}

	items := make([]voiceBinding, 0, len(payload))
	for _, item := range payload {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = id
		}
		providerVoice := strings.TrimSpace(firstNonEmpty(item.ProviderVoice, item.Voice, item.Name, item.ID))
		if providerVoice == "" {
			providerVoice = id
		}
		items = append(items, voiceBinding{
			VoiceOption: VoiceOption{
				ID:          id,
				Label:       label,
				Language:    strings.TrimSpace(item.Language),
				Description: strings.TrimSpace(item.Description),
				IsDefault:   item.IsDefault,
			},
			ProviderVoice: providerVoice,
		})
	}
	return normalizeVoiceBindings(items, fallbackDefault)
}

func normalizeVoiceBindings(items []voiceBinding, preferredDefault ...string) []voiceBinding {
	if len(items) == 0 {
		return nil
	}
	out := make([]voiceBinding, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	defaultID := ""
	for _, preferred := range preferredDefault {
		preferred = strings.TrimSpace(preferred)
		if preferred != "" {
			defaultID = preferred
			break
		}
	}
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Label = strings.TrimSpace(item.Label)
		item.Language = strings.TrimSpace(item.Language)
		item.Description = strings.TrimSpace(item.Description)
		item.ProviderVoice = strings.TrimSpace(item.ProviderVoice)
		if item.ID == "" {
			continue
		}
		if item.Label == "" {
			item.Label = item.ID
		}
		if item.ProviderVoice == "" {
			item.ProviderVoice = item.ID
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		if defaultID != "" && item.ID == defaultID {
			item.IsDefault = true
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil
	}

	hasDefault := false
	for i := range out {
		if defaultID != "" {
			out[i].IsDefault = out[i].ID == defaultID
		}
		if out[i].IsDefault {
			hasDefault = true
		}
	}
	if !hasDefault {
		out[0].IsDefault = true
	}
	return out
}

func bindingOptions(items []voiceBinding) []VoiceOption {
	if len(items) == 0 {
		return nil
	}
	out := make([]VoiceOption, 0, len(items))
	for _, item := range items {
		out = append(out, item.VoiceOption)
	}
	return out
}

func resolveVoiceBinding(requested string, voices []voiceBinding) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		for _, item := range voices {
			if item.ID == requested {
				return item.ProviderVoice, nil
			}
		}
		return "", ErrInvalidVoice
	}
	for _, item := range voices {
		if item.IsDefault {
			return item.ProviderVoice, nil
		}
	}
	if len(voices) > 0 {
		return voices[0].ProviderVoice, nil
	}
	return "", ErrInvalidVoice
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *Service) requestSpeech(ctx context.Context, text, voice string) ([]byte, string, error) {
	endpoint, err := joinEndpoint(s.cfg.apiBase, s.cfg.speechPath, "/audio/speech")
	if err != nil {
		return nil, "", err
	}

	reqBody := map[string]any{
		"model":           s.cfg.model,
		"voice":           voice,
		"input":           text,
		"format":          s.cfg.format,
		"response_format": s.cfg.format,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	if s.cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", contentTypeForFormat(s.cfg.format))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 240 {
			msg = msg[:240]
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, "", fmt.Errorf("tts provider http %d: %s", resp.StatusCode, msg)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = contentTypeForFormat(s.cfg.format)
	}
	return body, contentType, nil
}

func joinEndpoint(base, p, fallbackPath string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid tts api base")
	}
	targetPath := strings.TrimSpace(p)
	if targetPath == "" {
		targetPath = fallbackPath
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	u.Path = strings.TrimRight(u.Path, "/") + targetPath
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func contentTypeForFormat(format string) string {
	if strings.EqualFold(format, "wav") {
		return "audio/wav"
	}
	return "audio/mpeg"
}
