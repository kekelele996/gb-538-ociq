package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"industrial-noise-source-attribution/backend/internal/constants"
	"industrial-noise-source-attribution/backend/internal/model"
)

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		role       string
		permission string
		want       int
	}{
		{constants.RoleAcousticEngineer, PermissionSourceWrite, http.StatusNoContent},
		{constants.RoleDataAnalyst, PermissionSourceWrite, http.StatusForbidden},
		{constants.RoleReviewer, PermissionConfirm, http.StatusNoContent},
		{constants.RoleAuditor, PermissionRun, http.StatusForbidden},
	}
	for _, test := range tests {
		engine := gin.New()
		engine.Use(func(c *gin.Context) {
			c.Set("request_id", "rbac-test-request")
			c.Set("actor", model.Actor{ID: 7, Username: "fixture", DisplayName: "Fixture", Role: test.role})
		})
		engine.GET("/", RequirePermission(test.permission), func(c *gin.Context) { c.Status(http.StatusNoContent) })
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != test.want {
			t.Fatalf("role %s permission %s returned %d, want %d", test.role, test.permission, response.Code, test.want)
		}
	}
}
