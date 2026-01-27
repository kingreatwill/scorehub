package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type tokenPayload struct {
	UserID int64 `json:"uid"`
	Exp    int64 `json:"exp"`
}

func SignToken(secret []byte, userID int64, expiresIn time.Duration) (string, error) {
	if len(secret) == 0 {
		return "", errors.New("empty token secret")
	}
	p := tokenPayload{
		UserID: userID,
		Exp:    time.Now().Add(expiresIn).Unix(),
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(raw)
	sig := sign(secret, payload)
	return "sh1." + payload + "." + sig, nil
}

func ParseToken(secret []byte, token string) (int64, error) {
	if len(secret) == 0 {
		return 0, errors.New("empty token secret")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, errors.New("empty token")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != "sh1" {
		return 0, errors.New("invalid token format")
	}

	payloadPart := parts[1]
	sigPart := parts[2]
	wantSig := sign(secret, payloadPart)
	if subtle.ConstantTimeCompare([]byte(sigPart), []byte(wantSig)) != 1 {
		return 0, errors.New("invalid token signature")
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(payloadPart)
	if err != nil {
		return 0, fmt.Errorf("decode payload: %w", err)
	}

	var p tokenPayload
	if err := json.Unmarshal(payloadRaw, &p); err != nil {
		return 0, fmt.Errorf("parse payload: %w", err)
	}
	if p.UserID <= 0 {
		return 0, errors.New("invalid token user")
	}
	if p.Exp <= 0 || time.Now().Unix() > p.Exp {
		return 0, errors.New("token expired")
	}
	return p.UserID, nil
}

func sign(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

