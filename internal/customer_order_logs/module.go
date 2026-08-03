package customerorderlogs

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	logHandler "github.com/ganasa18/go-template/internal/customer_order_logs/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

// HTTPModule wires the customer-order automation logs endpoints.
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

// RegisterRoutes mounts the endpoints under /api/v1/customer-order-logs.
//
//	GET  /api/v1/customer-order-logs   list automation failure logs
//	POST /api/v1/customer-order-logs   create an automation failure log row
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	g := v1.Group("/customer-order-logs")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.GET("", m.base.RunAction(m.handler.List))
		g.POST("", m.base.RunAction(m.handler.Create))
	}
}
