package productiondashboard

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	productionDashboardHandler "github.com/ganasa18/go-template/internal/production_dashboard/handler"
	productionDashboardService "github.com/ganasa18/go-template/internal/production_dashboard/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *productionDashboardHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       productionDashboardService.IService
}

func NewHTTPModule(cfg *config.Config, base *baseHandler.BaseHTTPHandler, handler *productionDashboardHandler.HTTPHandler, authenticator authService.Authenticator, roleSvc roleService.IRoleService, svc productionDashboardService.IService) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator, roleService: roleSvc, service: svc}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}
	g := r.Group("/api/v1/production-dashboard")
	g.Use(auth)
	g.GET("/fg-dashboard", perm("production", "view"), m.base.RunAction(m.handler.FGDashboard))
	g.GET("/wip-dashboard", perm("production", "view"), m.base.RunAction(m.handler.WIPDashboard))
	g.GET("/output-machine-dashboard", perm("production", "view"), m.base.RunAction(m.handler.OutputMachineDashboard))
	g.GET("/summary-stroke-dashboard", perm("production", "view"), m.base.RunAction(m.handler.SummaryStrokeDashboard))
	g.GET("/runtime", perm("production", "view"), m.base.RunAction(m.handler.Runtime))
}
