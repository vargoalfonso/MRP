package middleware

import "github.com/gin-gonic/gin"

// Security sets hardening headers on every response.
func Security() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		// Cache-Control: prevent caching of API responses.
		c.Header("Cache-Control", "no-store")
		c.Header("Pragma", "no-cache")
		// Strict-Transport-Security is intentionally omitted here.
		// Set it at the load balancer / TLS termination layer in production.
		c.Next()
	}
}
