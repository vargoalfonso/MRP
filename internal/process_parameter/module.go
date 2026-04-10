package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	processHandler "github.com/ganasa18/go-template/internal/process_parameter/handler"
	processService "github.com/ganasa18/go-template/internal/process_parameter/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *processHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       processService.IProcessParameterService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *processHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service processService.IProcessParameterService,
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

	processGroup := v1.Group("/process")

	// 🔐 wajib login
	processGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		processGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "process", "view"), m.base.RunAction(m.handler.GetProcesses))
		processGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "process", "create"), m.base.RunAction(m.handler.CreateProcess))
		processGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "process", "view"), m.base.RunAction(m.handler.GetProcessByID))
		processGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "process", "update"), m.base.RunAction(m.handler.UpdateProcess))
		processGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "process", "delete"), m.base.RunAction(m.handler.DeleteProcess))
	}
}
