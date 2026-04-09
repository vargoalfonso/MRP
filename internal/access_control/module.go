package departement

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	acmHandler "github.com/ganasa18/go-template/internal/access_control/handler"
	acmService "github.com/ganasa18/go-template/internal/access_control/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *acmHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       acmService.IACMService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *acmHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service acmService.IACMService,
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

	userGroup := v1.Group("/user")

	// 🔐 wajib login
	userGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		userGroup.GET("", roleMiddleware.RequirePermission(m.roleService, "users", "view"), m.base.RunAction(m.handler.GetACMs))
		userGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "users", "create"), m.base.RunAction(m.handler.CreateACM))
		userGroup.GET("/:id", roleMiddleware.RequirePermission(m.roleService, "users", "view"), m.base.RunAction(m.handler.GetACMByID))
		userGroup.PUT("/:id", roleMiddleware.RequirePermission(m.roleService, "users", "update"), m.base.RunAction(m.handler.UpdateACM))
		userGroup.DELETE("/:id", roleMiddleware.RequirePermission(m.roleService, "users", "delete"), m.base.RunAction(m.handler.DeleteACM))
	}
}
