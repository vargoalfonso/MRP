package app

import (
	"net/http"

	"github.com/ganasa18/go-template/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Context wraps *gin.Context with application-level fields.
// request_id is guaranteed non-empty: it reads X-Request-Id from the inbound
// header (so upstream gateways can propagate IDs), falling back to a new UUIDv4.
// The ID is:
//   - Stored in Context.APIReqID for handler access
//   - Injected into request.Context() so all slog calls carry request_id
//   - Written to X-Request-Id response header for client correlation
type Context struct {
	*gin.Context
	Request  *http.Request
	APIReqID string
}

// NewContext builds an app.Context from a *gin.Context.
// Call this at the top of every RunAction so the request_id is set before any
// logging or downstream calls happen.
func NewContext(c *gin.Context) *Context {
	reqID := c.GetHeader("X-Request-Id")
	if reqID == "" {
		reqID = uuid.New().String()
	}

	// Inject into Go context so logger.FromContext() picks it up in all layers.
	ctx, _ := logger.WithRequestID(c.Request.Context(), reqID)
	c.Request = c.Request.WithContext(ctx)

	// Propagate to response header for client-side correlation.
	c.Header("X-Request-Id", reqID)

	return &Context{
		Context:  c,
		Request:  c.Request,
		APIReqID: reqID,
	}
}
