package qc

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	qcHandler "github.com/ganasa18/go-template/internal/qc/handler"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *qcHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *qcHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{cfg: cfg, base: base, handler: handler, authenticator: authenticator, roleService: roleSvc}
}

// RegisterRoutes registers QC endpoints.
// Base: /api/v1/qc
func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	g := r.Group("/api/v1/qc")
	g.Use(auth)

	tasks := g.Group("/tasks")
	tasks.GET("",
		roleMiddleware.RequirePermission(m.roleService, "qc", "view"),
		m.base.RunAction(m.handler.ListTasks),
	)
	tasks.POST("/:id/start",
		roleMiddleware.RequirePermission(m.roleService, "qc", "update"),
		m.base.RunAction(m.handler.StartTask),
	)
	tasks.POST("/:id/approve",
		roleMiddleware.RequirePermission(m.roleService, "qc", "update"),
		m.base.RunAction(m.handler.ApproveIncoming),
	)
	tasks.POST("/:id/reject",
		roleMiddleware.RequirePermission(m.roleService, "qc", "update"),
		m.base.RunAction(m.handler.RejectIncoming),
	)
}
