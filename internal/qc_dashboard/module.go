package qcdashboard

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	qcDashboardHandler "github.com/ganasa18/go-template/internal/qc_dashboard/handler"
	qcDashboardService "github.com/ganasa18/go-template/internal/qc_dashboard/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *qcDashboardHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       qcDashboardService.IService
}

func NewHTTPModule(cfg *config.Config, base *baseHandler.BaseHTTPHandler, handler *qcDashboardHandler.HTTPHandler, authenticator authService.Authenticator, roleSvc roleService.IRoleService, svc qcDashboardService.IService) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator, roleService: roleSvc, service: svc}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}
	g := r.Group("/api/v1/qc-dashboard")
	g.Use(auth)
	g.GET("/overview", perm("qc_dashboard", "view"), m.base.RunAction(m.handler.GetOverview))
	g.GET("/production-qc", perm("qc_dashboard", "view"), m.base.RunAction(m.handler.ListProductionQC))
	g.GET("/incoming-qc", perm("qc_dashboard", "view"), m.base.RunAction(m.handler.ListIncomingQC))
	g.GET("/defects", perm("qc_dashboard", "view"), m.base.RunAction(m.handler.ListDefects))
	g.GET("/issue-list", perm("qc_dashboard", "view"), m.base.RunAction(m.handler.ListIssueTypes))
	g.POST("/reports/manual", perm("qc_dashboard", "create"), m.base.RunAction(m.handler.CreateManualQCReport))
	g.POST("/defects/:defect_id/create-rework", perm("qc_dashboard", "create"), m.base.RunAction(m.handler.CreateReworkTask))
}
