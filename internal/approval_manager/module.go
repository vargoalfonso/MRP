package approvalmanager

import (
	"github.com/ganasa18/go-template/config"
	approvalHandler "github.com/ganasa18/go-template/internal/approval_manager/handler"
	approvalService "github.com/ganasa18/go-template/internal/approval_manager/service"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *approvalHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       approvalService.IService
}

func NewHTTPModule(cfg *config.Config, base *baseHandler.BaseHTTPHandler, handler *approvalHandler.HTTPHandler, authenticator authService.Authenticator, roleSvc roleService.IRoleService, svc approvalService.IService) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator, roleService: roleSvc, service: svc}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	g := r.Group("/api/v1/approval-manager")
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))
	{
		g.GET("/summary", roleMiddleware.RequirePermission(m.roleService, "approval_manager", "view"), m.base.RunAction(m.handler.GetSummary))
		g.GET("/items", roleMiddleware.RequirePermission(m.roleService, "approval_manager", "view"), m.base.RunAction(m.handler.ListItems))
		g.GET("/items/:id", roleMiddleware.RequirePermission(m.roleService, "approval_manager", "view"), m.base.RunAction(m.handler.GetDetail))
	}
}
