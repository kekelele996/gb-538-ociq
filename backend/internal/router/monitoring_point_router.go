package router

import (
	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/handler"
	"industrial-noise-source-attribution/backend/internal/middleware"
)

func RegisterMonitoringPointRoutes(api *gin.RouterGroup, value *handler.MonitoringPointHandler) {
	group := api.Group("/monitoring-points")
	group.GET("", middleware.RequirePermission(middleware.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(middleware.PermissionRead), value.Get)
	group.POST("", middleware.RequirePermission(middleware.PermissionPointWrite), value.Create)
	group.PUT("/:id", middleware.RequirePermission(middleware.PermissionPointWrite), value.Update)
	group.POST("/:id/deactivate", middleware.RequirePermission(middleware.PermissionPointWrite), value.Deactivate)
}

func RegisterSupportRoutes(v1, api *gin.RouterGroup, value *handler.SupportHandler, loginLimiter *middleware.RateLimiter) {
	v1.POST("/auth/login", loginLimiter.Middleware("login"), value.Login)
	api.GET("/audit-logs", middleware.RequirePermission(middleware.PermissionAudit), value.Audits)
	api.GET("/meta/enums", middleware.RequirePermission(middleware.PermissionRead), value.Enums)
}
