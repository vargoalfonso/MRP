package machinepattern

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	machineHandler "github.com/ganasa18/go-template/internal/machine_pattern/handler"
	machineService "github.com/ganasa18/go-template/internal/machine_pattern/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *machineHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       machineService.IService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *machineHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleSvc roleService.IRoleService,
	svc machineService.IService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleSvc,
		service:       svc,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	auth := authMiddleware.JWTMiddleware(m.authenticator)
	perm := func(resource, action string) gin.HandlerFunc {
		return roleMiddleware.RequirePermission(m.roleService, resource, action)
	}

	v1 := r.Group("/api/v1")
	{
		g := v1.Group("/machine-patterns")
		g.Use(auth)
		g.GET("", perm("machine_pattern", "view"), m.base.RunAction(m.handler.GetMachinePatterns))
		g.POST("", perm("machine_pattern", "create"), m.base.RunAction(m.handler.CreateMachinePattern))
		g.GET("/summary", perm("machine_pattern", "view"), m.base.RunAction(m.handler.GetMachinePatternSummary))
		g.GET("/:id", perm("machine_pattern", "view"), m.base.RunAction(m.handler.GetMachinePatternByID))
		g.PUT("/:id", perm("machine_pattern", "update"), m.base.RunAction(m.handler.UpdateMachinePattern))
		g.DELETE("/:id", perm("machine_pattern", "delete"), m.base.RunAction(m.handler.DeleteMachinePattern))
		g.POST("/bulk", perm("machine_pattern", "create"), m.base.RunAction(m.handler.BulkCreateMachinePattern))
	}
}
