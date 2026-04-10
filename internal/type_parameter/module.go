package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	typeParameter "github.com/ganasa18/go-template/internal/type_parameter/handler"
	typeParameterService "github.com/ganasa18/go-template/internal/type_parameter/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *typeParameter.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       typeParameterService.ITypeService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *typeParameter.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service typeParameterService.ITypeService,
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

	typeParameterGroup := v1.Group("/type-parameter")

	// 🔐 wajib login
	typeParameterGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		typeParameterGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "type-parameter", "view"), m.base.RunAction(m.handler.GetAll))
		typeParameterGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "type-parameter", "create"), m.base.RunAction(m.handler.Create))
		typeParameterGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "type-parameter", "view"), m.base.RunAction(m.handler.GetByID))
		typeParameterGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "type-parameter", "update"), m.base.RunAction(m.handler.Update))
		typeParameterGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "type-parameter", "delete"), m.base.RunAction(m.handler.Delete))
	}
}
