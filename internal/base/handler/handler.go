package handler

import (
	"log/slog"
	"time"

	"github.com/ganasa18/go-template/internal/base/app"
	"github.com/ganasa18/go-template/pkg/logger"
	"github.com/ganasa18/go-template/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HTTPHandlerFunc is the standard handler signature used across all domains.
type HTTPHandlerFunc func(ctx *app.Context) *app.CostumeResponse

// BaseHTTPHandler carries shared infrastructure and provides the RunAction wrapper.
type BaseHTTPHandler struct {
	DB *gorm.DB
}

// NewBaseHTTPHandler constructs a BaseHTTPHandler.
func NewBaseHTTPHandler(db *gorm.DB) *BaseHTTPHandler {
	return &BaseHTTPHandler{DB: db}
}

// RunAction wraps an HTTPHandlerFunc with:
//   - app.NewContext (request_id generation, context injection, X-Request-Id header)
//   - structured request/response logging via slog
//   - panic recovery (no internal details leaked to client)
func (h *BaseHTTPHandler) RunAction(handler HTTPHandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx := app.NewContext(c)
		log := logger.FromContext(ctx.Request.Context())

		log.Info("request received",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)

		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					slog.Any("error", err),
					slog.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(500, &app.CostumeResponse{
					RequestID: ctx.APIReqID,
					Status:    500,
					Message:   "request halted unexpectedly, please contact the administrator",
				})
			}
		}()

		resp := handler(ctx)
		latency := time.Since(start).Milliseconds()

		isError := resp.Status >= 400
		if isError {
			log.Error("request error",
				slog.String("path", c.Request.URL.Path),
				slog.Int("status", resp.Status),
				slog.String("message", resp.Message),
				slog.Int64("latency_ms", latency),
			)
		} else {
			log.Info("request completed",
				slog.Int("status", resp.Status),
				slog.String("path", c.Request.URL.Path),
				slog.Int64("latency_ms", latency),
			)
		}

		c.JSON(resp.Status, resp)
	}
}

// AbortWithError is a helper for middleware that needs to short-circuit before
// an app.Context exists (e.g., JWT middleware). It uses pkg/response.Abort.
func AbortWithError(c *gin.Context, err error) {
	response.Abort(c, err)
}
