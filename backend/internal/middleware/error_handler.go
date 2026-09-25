package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func ErrorHandler(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		logger.Error("request recorded an internal error",
			"request_id", c.GetString("request_id"), "method", c.Request.Method,
			"path", c.Request.URL.Path, "status", c.Writer.Status(), "error", c.Errors.Last().Err.Error(),
		)
	}
}
