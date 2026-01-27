package middleware

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

func RequestLog() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)

		status := c.Response.StatusCode()
		method := string(c.Method())
		path := string(c.Path())

		uri := path
		if q := strings.TrimSpace(string(c.Request.URI().QueryString())); q != "" {
			uri = uri + "?" + sanitizeQuery(q)
		}

		ip := c.ClientIP()

		uid := "-"
		if v, ok := c.Get(ctxUserIDKey); ok {
			if id, ok := v.(int64); ok {
				uid = strconv.FormatInt(id, 10)
			}
		}

		cost := time.Since(start)
		errStr := strings.TrimSpace(c.Errors.String())
		if errStr != "" {
			errStr = strings.ReplaceAll(errStr, "\n", " | ")
			log.Printf("[REQ] %d %s %s cost=%s ip=%s uid=%s err=%s", status, method, uri, cost, ip, uid, errStr)
			return
		}
		log.Printf("[REQ] %d %s %s cost=%s ip=%s uid=%s", status, method, uri, cost, ip, uid)
	}
}

func sanitizeQuery(raw string) string {
	parts := strings.Split(raw, "&")
	for i, p := range parts {
		if p == "" {
			continue
		}
		k, v, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "token", "access_token", "authorization", "code":
			parts[i] = k + "=***"
		default:
			parts[i] = k + "=" + v
		}
	}
	return strings.Join(parts, "&")
}
