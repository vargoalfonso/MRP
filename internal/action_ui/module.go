package action_ui

import (
	"github.com/ganasa18/go-template/config"
	actionHandler "github.com/ganasa18/go-template/internal/action_ui/handler"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *actionHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *actionHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleSvc,
	}
}

// RegisterRoutes registers Action UI endpoints.
// Base: /api/v1/action-ui
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)

	g := r.Group("/api/v1/action-ui")
	g.Use(auth)

	incoming := g.Group("/incoming")
	incoming.POST("/scans",
		roleMiddleware.RequirePermission(m.roleService, "action_ui", "create"),
		m.base.RunAction(m.handler.CreateIncomingScan),
	)
}
