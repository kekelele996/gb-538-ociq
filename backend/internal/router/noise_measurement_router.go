package router

import (
	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/handler"
	"industrial-noise-source-attribution/backend/internal/middleware"
)

func RegisterNoiseMeasurementRoutes(api *gin.RouterGroup, value *handler.NoiseMeasurementHandler, importLimiter *middleware.RateLimiter) {
	group := api.Group("/noise-measurements")
	group.GET("", middleware.RequirePermission(middleware.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(middleware.PermissionRead), value.Get)
	group.POST("", middleware.RequirePermission(middleware.PermissionImport), importLimiter.Middleware("measurement-import"), value.Create)
	group.POST("/:id/transition", middleware.RequirePermission(middleware.PermissionMeasureFlow), value.Transition)
}
