package import_file

import (
	"github.com/ganasa18/go-template/config"
	authMiddleware "github.com/ganasa18/go-template/internal/auth/middleware"
	authService "github.com/ganasa18/go-template/internal/auth/service"
	baseHandler "github.com/ganasa18/go-template/internal/base/handler"
	appmodule "github.com/ganasa18/go-template/internal/module"

	importHandler "github.com/ganasa18/go-template/internal/import_file/handler"
	importService "github.com/ganasa18/go-template/internal/import_file/service"
	roleMiddleware "github.com/ganasa18/go-template/internal/role/middleware"
	roleService "github.com/ganasa18/go-template/internal/role/service"

	"github.com/gin-gonic/gin"
)

var _ appmodule.HTTPModule = (*HTTPModule)(nil)

type HTTPModule struct {
	cfg           *config.Config
	base          *baseHandler.BaseHTTPHandler
	handler       *importHandler.HTTPHandler
	authenticator authService.Authenticator
	roleService   roleService.IRoleService
	service       importService.ImportService
}

func NewHTTPModule(
	cfg *config.Config,
	base *baseHandler.BaseHTTPHandler,
	handler *importHandler.HTTPHandler,
	authenticator authService.Authenticator,
	roleService roleService.IRoleService,
	service importService.ImportService,
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

	userGroup := v1.Group("/import")

	// 🔐 wajib login
	userGroup.Use(authMiddleware.JWTMiddleware(m.authenticator))

	{
		userGroup.POST("", roleMiddleware.RequirePermission(m.roleService, "users", "create"), m.base.RunAction(m.handler.ImportExcel))
	}
}
