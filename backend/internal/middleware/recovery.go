package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"industrial-noise-source-attribution/backend/internal/util"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered", "request_id", c.GetString("request_id"), "path", c.Request.URL.Path, "panic", fmt.Sprint(recovered))
		util.RespondError(c, util.NewError(http.StatusInternalServerError, "internal_error", "服务暂时无法完成请求", nil))
		c.Abort()
	})
}
