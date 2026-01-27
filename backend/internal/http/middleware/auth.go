package middleware

import (
	"context"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	appconfig "scorehub/internal/config"
	"scorehub/internal/auth"
	"scorehub/internal/store"
)

const ctxUserIDKey = "scorehub.userID"

func AuthRequired(cfg appconfig.Config, st *store.Store) app.HandlerFunc {
	secret := []byte(cfg.TokenSecret)
	return func(ctx context.Context, c *app.RequestContext) {
		token := extractBearerToken(string(c.GetHeader("Authorization")))
		if token == "" && cfg.DevAuth {
			if openid := strings.TrimSpace(string(c.GetHeader("X-Dev-OpenID"))); openid != "" {
				u, err := st.UpsertUserByOpenID(ctx, openid, "", "")
				if err == nil {
					c.Set(ctxUserIDKey, u.ID)
					c.Next(ctx)
					return
				}
			}
		}

		if token == "" {
			c.AbortWithStatusJSON(401, map[string]any{
				"error": map[string]any{"code": "unauthorized", "message": "missing token"},
			})
			return
		}

		uid, err := auth.ParseToken(secret, token)
		if err != nil {
			c.AbortWithStatusJSON(401, map[string]any{
				"error": map[string]any{"code": "unauthorized", "message": "invalid token"},
			})
			return
		}
		c.Set(ctxUserIDKey, uid)
		c.Next(ctx)
	}
}

func UserID(c *app.RequestContext) (int64, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

func extractBearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}
	const prefix = "Bearer "
	if strings.HasPrefix(authHeader, prefix) {
		return strings.TrimSpace(authHeader[len(prefix):])
	}
	return ""
}

