package customerorder

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	coHandler "github.com/ganasa18/go-template/internal/customer_order/handler"
	coService "github.com/ganasa18/go-template/internal/customer_order/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *coHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       coService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *coHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service coService.IService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleService,
		service:       service,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")
	g := v1.Group("/customer-orders")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.POST("", roleMiddleware.RequirePermission(m.roleService, "customer_order", "create"), m.base.RunAction(m.handler.Create))
		g.GET("", roleMiddleware.RequirePermission(m.roleService, "customer_order", "view"), m.base.RunAction(m.handler.List))
		g.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "customer_order", "view"), m.base.RunAction(m.handler.GetByID))
		g.PATCH("/:id/status", roleMiddleware.RequirePermission(m.roleService, "customer_order", "update"), m.base.RunAction(m.handler.UpdateStatus))
		g.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "customer_order", "delete"), m.base.RunAction(m.handler.Delete))
	}
}
