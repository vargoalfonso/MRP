package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ganasa18/go-template/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Recovery replaces gin's built-in recovery. It logs the panic with the
// request_id from context (set by app.NewContext) and returns a sanitised
// 500 — no stack traces or internal details are exposed to the client.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log := logger.FromContext(c.Request.Context())
				log.Error("panic recovered",
					slog.Any("error", err),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success":    false,
					"message":    "request halted unexpectedly, please contact the administrator",
					"request_id": c.GetHeader("X-Request-Id"),
				})
			}
		}()
		c.Next()
	}
}
