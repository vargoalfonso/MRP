package middleware

import (
	"net/http"

	"github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsRaw, exists := c.Get("claims")
		if !exists {
			c.AbortWithStatusJSON(401, gin.H{"message": "unauthorized"})
			return
		}

		claims, ok := claimsRaw.(*models.Claims)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"message": "invalid claims"})
			return
		}

		for _, userRole := range claims.Roles {
			for _, allowed := range allowedRoles {
				if userRole == allowed {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(403, gin.H{
			"message": "forbidden",
		})
	}
}

func RequirePermission(roleService service.IRoleService, feature, action string) gin.HandlerFunc {
	return func(c *gin.Context) {

		appCtx := app.NewContext(c)

		claimsRaw, exists := c.Get("claims")
		if !exists {
			res := &app.CostumeResponse{
				RequestID: appCtx.APIReqID,
				Status:    http.StatusUnauthorized,
				Message:   "unauthorized",
			}
			c.AbortWithStatusJSON(res.Status, res)
			return
		}

		claims := claimsRaw.(*models.Claims)

		// ambil role dari JWT
		roleName := claims.Roles[0]

		// 🔥 ambil dari DB
		permissions, err := roleService.GetPermissions(c.Request.Context(), roleName)
		if err != nil {
			res := &app.CostumeResponse{
				RequestID: appCtx.APIReqID,
				Status:    http.StatusInternalServerError,
				Message:   "failed to load permissions",
			}
			c.AbortWithStatusJSON(res.Status, res)
			return
		}

		featureMap, ok := permissions[feature].(map[string]interface{})
		if !ok {
			res := &app.CostumeResponse{
				RequestID: appCtx.APIReqID,
				Status:    http.StatusForbidden,
				Message:   "forbidden",
			}
			c.AbortWithStatusJSON(res.Status, res)
			return
		}

		allowed, ok := featureMap[action].(bool)
		if !ok || !allowed {
			res := &app.CostumeResponse{
				RequestID: appCtx.APIReqID,
				Status:    http.StatusForbidden,
				Message:   "forbidden",
			}
			c.AbortWithStatusJSON(res.Status, res)
			return
		}

		c.Next()
	}
}
