package mechin_parameter

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	mechinHandler "github.com/ganasa18/go-template/internal/mechin_parameter/handler"
	mechinService "github.com/ganasa18/go-template/internal/mechin_parameter/service"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *mechinHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       mechinService.IMechinParameterService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *mechinHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service mechinService.IMechinParameterService,
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

	mechinParameterGroup := v1.Group("/machine-parameter")
	mechinParameterGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		mechinParameterGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "machine_parameter", "view"), m.base.RunAction(m.handler.GetMechinParameters))
		mechinParameterGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "machine_parameter", "create"), m.base.RunAction(m.handler.CreateMechinParameter))
		mechinParameterGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "machine_parameter", "view"), m.base.RunAction(m.handler.GetMechinParameterByID))
		mechinParameterGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "machine_parameter", "update"), m.base.RunAction(m.handler.UpdateMechinParameter))
		mechinParameterGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "machine_parameter", "delete"), m.base.RunAction(m.handler.DeleteMechinParameter))
	}
}
