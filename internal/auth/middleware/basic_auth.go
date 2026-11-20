package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BasicAuthMiddleware validates an Authorization: Basic <base64> header against
// the given username and password. Uses constant-time comparison to prevent
// timing-based credential discovery.
//
// Intended for machine-to-machine endpoints (e.g. cron-triggered admin jobs)
// where JWT is inconvenient.
func BasicAuthMiddleware(user, pass string) gin.HandlerFunc {
	return func(c *gin.Context) {
		gotUser, gotPass, ok := c.Request.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(gotUser), []byte(user)) != 1 ||
			subtle.ConstantTimeCompare([]byte(gotPass), []byte(pass)) != 1 {
			c.Header("WWW-Authenticate", `Basic realm="admin-jobs"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  http.StatusUnauthorized,
				"message": "invalid credentials",
			})
			return
		}
		c.Next()
	}
}
