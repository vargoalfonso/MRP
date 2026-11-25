package mastermachine

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	machineHandler "github.com/ganasa18/go-template/internal/master_machine/handler"
	machineService "github.com/ganasa18/go-template/internal/master_machine/service"
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
	service       machineService.IMasterMachineService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *machineHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service machineService.IMasterMachineService,
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

	g := v1.Group("/machines")

	// wajib login
	g.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		g.GET("", roleMiddleware.RequirePermission(m.roleService, "machine", "view"), m.base.RunAction(m.handler.GetMachines))
		g.POST("", roleMiddleware.RequirePermission(m.roleService, "machine", "create"), m.base.RunAction(m.handler.CreateMachine))
		g.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "machine", "view"), m.base.RunAction(m.handler.GetMachineByID))
		g.GET("/:id/qr", roleMiddleware.RequirePermission(m.roleService, "machine", "view"), m.base.RunAction(m.handler.GetMachineQR))
		g.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "machine", "update"), m.base.RunAction(m.handler.UpdateMachine))
		g.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "machine", "delete"), m.base.RunAction(m.handler.DeleteMachine))
	}
}
