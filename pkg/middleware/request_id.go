// Package middleware contains all reusable Gin middleware for the application.
package middleware

import (
	"github.com/ganasa18/go-template/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const headerRequestID = "X-Request-Id"

// RequestID generates a UUIDv4 for every inbound request.
// It:
//   - Reads an existing X-Request-Id header (useful for trace propagation from gateways)
//   - Falls back to generating a new UUIDv4 if the header is absent or empty
//   - Injects the ID into request context (via logger.WithRequestID) so all
//     downstream log calls automatically carry the request_id field
//   - Sets the X-Request-Id response header so clients can correlate logs
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(headerRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx, _ := logger.WithRequestID(c.Request.Context(), reqID)
		c.Request = c.Request.WithContext(ctx)

		c.Header(headerRequestID, reqID)
		c.Next()
	}
}
