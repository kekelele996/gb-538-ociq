package router

import (
	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/handler"
	"industrial-noise-source-attribution/backend/internal/middleware"
)

func RegisterAttributionRunRoutes(api *gin.RouterGroup, value *handler.AttributionRunHandler, limiter *middleware.RateLimiter) {
	group := api.Group("/attribution-runs")
	group.GET("", middleware.RequirePermission(middleware.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(middleware.PermissionRead), value.Get)
	group.GET("/:id/compare/:other_id", middleware.RequirePermission(middleware.PermissionRead), value.Compare)
	group.POST("", middleware.RequirePermission(middleware.PermissionRun), limiter.Middleware("attribution-run"), value.Create)
	group.POST("/:id/review", middleware.RequirePermission(middleware.PermissionReview), value.Review)
	group.POST("/:id/confirm", middleware.RequirePermission(middleware.PermissionConfirm), value.Confirm)
	group.POST("/:id/void", middleware.RequirePermission(middleware.PermissionVoid), value.Void)
}
