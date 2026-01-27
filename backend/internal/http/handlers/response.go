package handlers

import (
	"github.com/cloudwego/hertz/pkg/app"
)

func writeError(c *app.RequestContext, status int, code string, message string, errs ...error) {
	if len(errs) > 0 && errs[0] != nil {
		c.Error(errs[0])
	}
	c.JSON(status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
