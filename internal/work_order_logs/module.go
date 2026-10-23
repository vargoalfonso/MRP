package workorderlogs

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	logHandler "github.com/ganasa18/go-template/internal/work_order_logs/handler"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

// HTTPModule wires the work-order automation logs endpoints.
type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *logHandler.HTTPHandler
	authenticator authService.Authenticator
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *logHandler.HTTPHandler,
	authenticator authService.Authenticator,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
	}
}

// RegisterRoutes mounts the endpoints under /api/v1/work-order-logs.
//
//	GET  /api/v1/work-order-logs   list automation failure logs (filter: wo_number, automation_job_id, search)
//	POST /api/v1/work-order-logs   create an automation failure log row
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	g := v1.Group("/work-order-logs")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.GET("", m.base.RunAction(m.handler.List))
		g.POST("", m.base.RunAction(m.handler.Create))
	}
}
