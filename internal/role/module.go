package role

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"
	roleHandler "github.com/ganasa18/go-template/internal/role/handler"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"
	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *roleHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *roleHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
) appmodule.HTTPModule {
	return &HTTPModule{
		cfg:           cfg,
		base:          base,
		handler:       handler,
		authenticator: authenticator,
		roleService:   roleService,
	}
}

func (m *HTTPModule) RegisterRoutes(r gin.IRouter) {
	v1 := r.Group("/api/v1")

	roleGroup := v1.Group("/roles")

	// 🔐 WAJIB LOGIN
	roleGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		roleGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "role", "view"), m.base.RunAction(m.handler.GetRoles))
		roleGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "role", "create"), m.base.RunAction(m.handler.CreateRole))
		roleGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "role", "view"), m.base.RunAction(m.handler.GetRoleByID))
		roleGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "role", "update"), m.base.RunAction(m.handler.UpdateRole))
		roleGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "role", "delete"), m.base.RunAction(m.handler.DeleteRole))
	}
}
