package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	globalParameterHandler "github.com/ganasa18/go-template/internal/global_parameter/handler"
	globalParameterService "github.com/ganasa18/go-template/internal/global_parameter/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *globalParameterHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       globalParameterService.IGlobalParameterService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *globalParameterHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service globalParameterService.IGlobalParameterService,
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

	globalParameterGroup := v1.Group("/global-parameters")

	// 🔐 wajib login
	globalParameterGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		globalParameterGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "global_parameter", "view"), m.base.RunAction(m.handler.GetGlobalParameters))
		globalParameterGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "global_parameter", "create"), m.base.RunAction(m.handler.CreateGlobalParameter))
		globalParameterGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "global_parameter", "view"), m.base.RunAction(m.handler.GetGlobalParameterByID))
		globalParameterGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "global_parameter", "update"), m.base.RunAction(m.handler.UpdateGlobalParameter))
		globalParameterGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "global_parameter", "delete"), m.base.RunAction(m.handler.DeleteGlobalParameter))
	}
}
