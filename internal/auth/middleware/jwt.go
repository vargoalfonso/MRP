// Package middleware provides the JWT authentication middleware for Gin.
package middleware

import (
	"strings"

	authService "github.com/ganasa18/go-template/internal/auth/service"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/response"
	"github.com/gin-gonic/gin"
)

const bearerPrefix = "Bearer "

// JWTMiddleware validates the Authorization: Bearer <token> header.
// On success it stores the *models.Claims under the key "claims" in gin.Context
// so downstream handlers can retrieve the authenticated user's ID and roles.
//
// Usage:
//
//	protected := router.Group("/api")
//	protected.Use(middleware.JWTMiddleware(auth))
func JWTMiddleware(auth authService.Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
			response.Abort(c, apperror.Unauthorized("missing or malformed Authorization header"))
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, bearerPrefix)
		claims, err := auth.ValidateAccessToken(c.Request.Context(), tokenStr)
		if err != nil {
			response.Abort(c, err)
			return
		}

		// Inject claims so handlers can access UserID / Roles without re-parsing.
		c.Set("claims", claims)
		c.Next()
	}
}
