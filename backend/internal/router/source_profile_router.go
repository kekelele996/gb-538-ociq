package router

import (
	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/handler"
	"industrial-noise-source-attribution/backend/internal/middleware"
)

func RegisterSourceProfileRoutes(api *gin.RouterGroup, value *handler.SourceProfileHandler) {
	group := api.Group("/source-profiles")
	group.GET("", middleware.RequirePermission(middleware.PermissionRead), value.List)
	group.GET("/:id", middleware.RequirePermission(middleware.PermissionRead), value.Get)
	group.POST("", middleware.RequirePermission(middleware.PermissionSourceWrite), value.Create)
	group.POST("/:id/transition", middleware.RequirePermission(middleware.PermissionSourceWrite), value.Transition)
}
