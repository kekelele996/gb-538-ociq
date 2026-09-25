package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/util"
)

const (
	PermissionRead        = "read"
	PermissionPointWrite  = "point:write"
	PermissionImport      = "measurement:import"
	PermissionMeasureFlow = "measurement:flow"
	PermissionSourceWrite = "source:write"
	PermissionRun         = "attribution:run"
	PermissionReview      = "attribution:review"
	PermissionConfirm     = "attribution:confirm"
	PermissionVoid        = "attribution:void"
	PermissionAudit       = "audit:read"
)

var permissions = map[string]map[string]bool{
	constants.RoleAdmin: {
		PermissionRead: true, PermissionPointWrite: true, PermissionImport: true,
		PermissionMeasureFlow: true, PermissionSourceWrite: true, PermissionRun: true,
		PermissionReview: true, PermissionConfirm: true, PermissionVoid: true, PermissionAudit: true,
	},
	constants.RoleAcousticEngineer: {
		PermissionRead: true, PermissionPointWrite: true, PermissionImport: true,
		PermissionMeasureFlow: true, PermissionSourceWrite: true, PermissionRun: true,
	},
	constants.RoleDataAnalyst: {
		PermissionRead: true, PermissionImport: true, PermissionMeasureFlow: true, PermissionRun: true,
	},
	constants.RoleReviewer: {PermissionRead: true, PermissionReview: true, PermissionConfirm: true, PermissionVoid: true, PermissionAudit: true},
	constants.RoleAuditor:  {PermissionRead: true, PermissionAudit: true},
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor, ok := ActorFromContext(c)
		if !ok {
			util.RespondError(c, util.NewError(http.StatusUnauthorized, "unauthorized", "认证上下文缺失", nil))
			c.Abort()
			return
		}
		if !permissions[actor.Role][permission] {
			util.RespondError(c, util.Forbidden("当前角色没有执行此操作的权限"))
			c.Abort()
			return
		}
		c.Next()
	}
}
